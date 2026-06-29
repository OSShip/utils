package observability

// currentService is set by InitLogger and attached to every log / Sentry event.
var currentService string

// CurrentService returns the service name passed to InitLogger.
func CurrentService() string {
	return currentService
}

func setCurrentService(name string) {
	currentService = name
}
