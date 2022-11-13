package main

import (
	"fmt"

	"net/http"
	"os"

	"github.com/datreeio/admission-webhook-datree/pkg/controllers"
	"github.com/datreeio/admission-webhook-datree/pkg/errorReporter"
	"github.com/datreeio/admission-webhook-datree/pkg/k8sMetadataUtil"
	"github.com/datreeio/admission-webhook-datree/pkg/loggerUtil"
	"github.com/datreeio/admission-webhook-datree/pkg/server"
	"github.com/datreeio/datree/pkg/cliClient"
	"github.com/datreeio/datree/pkg/deploymentConfig"
	"github.com/datreeio/datree/pkg/localConfig"
	"github.com/datreeio/datree/pkg/networkValidator"
	"github.com/datreeio/datree/pkg/printer"
	"github.com/datreeio/datree/pkg/utils"
)

const DefaultErrExitCode = 1

func main() {
	port := os.Getenv("LISTEN_PORT")
	if port == "" {
		port = "8443"
	}
	start(port)
}

func start(port string) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			loggerUtil.Log("panic error: " + fmt.Sprint(panicErr))
			globalPrinter := printer.CreateNewPrinter()
			validator := networkValidator.NewNetworkValidator()
			newCliClient := cliClient.NewCliClient(deploymentConfig.URL, validator)
			newLocalConfigClient := localConfig.NewLocalConfigClient(newCliClient, validator)
			reporter := errorReporter.NewErrorReporter(newCliClient, newLocalConfigClient)
			reporter.ReportPanicError(panicErr)
			globalPrinter.PrintMessage(fmt.Sprintf("Datree Webhook failed to start due to Unexpected error: %s\n", utils.ParseErrorToString(panicErr)), "error")
			os.Exit(DefaultErrExitCode)
		}
	}()

	loggerUtil.Log("initializing k8s metadata")
	k8sMetadataUtil.InitK8sMetadataUtil()

	certPath, keyPath, err := server.ValidateCertificate()
	if err != nil {
		panic(err)
	}

	validationController := controllers.NewValidationController()
	healthController := controllers.NewHealthController()
	// set routes
	http.HandleFunc("/validate", validationController.Validate)
	http.HandleFunc("/health", healthController.Health)
	http.HandleFunc("/ready", healthController.Ready)

	// start server
	err = http.ListenAndServeTLS(":"+port, certPath, keyPath, nil)
	if err != nil {
		loggerUtil.Log(err.Error())
		http.ListenAndServe(":"+port, nil)
	}
}