package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alexliesenfeld/health"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"time"
)

// This is a an example configuration for Kubernetes liveness and readiness checks (for more info, please refer to
// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).
// Please note that Kubernetes readiness and especially liveness checks need to be designed with care to not cause
// any unintended behaviour (such as unexpected pod restarts, cascading failures, etc.). Please refer to the following
// articles for guidance:
// - https://www.innoq.com/en/blog/kubernetes-probes/
// - https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-how-to-avoid-shooting-yourself-in-the-foot/
// - https://srcco.de/posts/kubernetes-liveness-probes-are-dangerous.html
// Attention: Please see file `example-pod-config.yaml` in the same directory for an example configuration
// that you can use to complement this check implementation example.
func main() {
	db, _ := sql.Open("sqlite3", "simple.sqlite")
	defer db.Close()

	// Create a new Checker for our readiness check.
	readinessChecker := health.NewChecker(

		// Configure a global timeout that will be applied to all check functions.
		health.WithTimeout(10*time.Second),

		// A check configuration to see if our database connection is up.
		// Be wary though that this should be a "service private" database instance.
		// If many of your services use the same database instance, the readiness checks
		// of all of these services will start failing on every small database hick-up.
		// This is most likely not what you want. For guidance, please refer to the links
		// listed in the main function documentation above.
		health.WithCheck(health.Check{
			Name:  "database", // A unique check name.
			Check: db.PingContext,
		}),

		// The following check will be executed periodically every 15 seconds
		// started with an initial delay of 3 seconds. The check function will NOT
		// be executed for each HTTP request.
		health.WithPeriodicCheck(15*time.Second, 3*time.Second, health.Check{
			Name: "search",
			// The check function checks the health of a component. If an error is
			// returned, the component is considered unavailable ("down").
			// The context contains a deadline according to the configuration of
			// the Checker (global and .
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this makes the check fail")
			},
		}),

		// Set a status listener that will be invoked when the health status changes.
		// More powerful hooks are also available (see docs). For guidance, please refer to the links
		// listed in the main function documentation above.
		health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
			log.Println(fmt.Sprintf("health status changed to %s", state.Status))
		}),
	)

	// Liveness check should mostly contain checks that identify if the service is locked up or in a state that it
	// cannot recover from (deadlocks, etc.). In most cases it should just respond with 200 OK to avoid unexpected
	// restarts.
	livenessChecker := health.NewChecker()

	// Create a new health check http.Handler that returns the health status
	// serialized as a JSON string. You can pass pass further configuration
	// options to NewHandler to modify default configuration.
	http.Handle("/live", health.NewHandler(livenessChecker))
	http.Handle("/ready", health.NewHandler(readinessChecker))

	// Start the HTTP server
	log.Fatalln(http.ListenAndServe(":3000", nil))
}
