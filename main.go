package main

import (
	"context"
	"net/http"
	"fmt"
	"os"

	"github.com/go-chi/chi"
	// "github.com/go-chi/chi/middleware"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"go.opentelemetry.io/otel/trace"
	"github.com/signalfx/splunk-otel-go/distro"
	"github.com/signalfx/splunk-otel-go/instrumentation/github.com/go-chi/chi/splunkchi"
)

func withTraceMetadata(ctx context.Context, logger logr.Logger) logr.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		// ctx does not contain a valid span.
		// There is no trace metadata to add.
		return logger
	}
	return logger.WithValues(
		"trace_id", spanContext.TraceID().String(),
		"span_id", spanContext.SpanID().String(),
		"trace_flags", spanContext.TraceFlags().String(),
	)
}

func helloHandler(logger logr.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//  logger.Printf("request context: %+v", ctx)
		l := withTraceMetadata(ctx, logger)

		// Log the trace metadata
		logger.Info("request context logged",
			"trace_id", trace.SpanContextFromContext(ctx).TraceID().String(),
			"span_id", trace.SpanContextFromContext(ctx).SpanID().String(),
			"trace_flags", trace.SpanContextFromContext(ctx).TraceFlags().String(),
		)

		n, err := w.Write([]byte("Hello World!\n"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error(err, "failed to write request response")
		} else {
			l.Info("request handled", "response_bytes", n)
		}
	}
}

func main() {

	// Set OTEL_EXPORTER_OTLP_ENDPOINT environment variable
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
	os.Setenv("OTEL_SERVICE_NAME", "golang-logging")
	os.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.version=1.0,deployment.environment=dev")

	// Retrieve the value of OTEL_EXPORTER_OTLP_ENDPOINT environment variable
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Print the value using fmt.Println or fmt.Printf
	fmt.Printf("OTEL_EXPORTER_OTLP_ENDPOINT: %s\n", endpoint)


	sdk, err := distro.Run()
	if err != nil {
		panic(err)
	}
	
	// Print a message after distro.Run() to ensure it's executed
	fmt.Println("OpenTelemetry SDK initialized successfully")

	// Flush all spans before the application exits
	defer func() {
		if err := sdk.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	logger := stdr.New(nil) // Use nil to use a no-op logger by default

	router := chi.NewRouter()
	//router.Use(middleware.RequestID)
	router.Use(splunkchi.Middleware())
	router.Get("/hello", helloHandler(logger))

	if err := http.ListenAndServe(":8080", router); err != nil {
		logger.Error(err, "failed to start server")
	}
}

