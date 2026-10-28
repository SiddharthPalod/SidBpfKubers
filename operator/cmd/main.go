package main

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
	"sidkubers.io/operator/internal/controller"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(databasev1alpha1.AddToScheme(scheme))
}

func main() {
	fmt.Println("==> Initializing DatabaseCluster Operator...")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0", // disables metrics port :8080 to avoid local port conflicts
		},
		LeaderElection:   true,
		LeaderElectionID: "database-operator-leader.sidkubers.io",
	})
	if err != nil {
		fmt.Printf("ERROR: unable to start manager: %v\n", err)
		os.Exit(1)
	}

	if err = (&controller.DatabaseClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		fmt.Printf("ERROR: unable to create controller: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("==> Manager started with Leader Election! Watching events (Press Ctrl+C to stop)...")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Printf("ERROR: problem running manager: %v\n", err)
		os.Exit(1)
	}
}
