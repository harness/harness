# Airbrake Hook for Logrus <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:" />

Use this hook to send your errors to [Airbrake](https://airbrake.io/).
This hook is using the [official airbrake go package](https://github.com/airbrake/gobrake), and will hit the api V3.
The hook is async for `log.Error`, but blocking for the notice to be sent with `log.Fatal` and `log.Panic`.

All logrus fields will be sent as context fields on Airbrake.

## Usage

The hook must be configured with:

* A project ID (found in your your Airbrake project settings)
* An API key ID (found in your your Airbrake project settings)
* The name of the current environment ("development", "staging", "production", ...)

```go
import (
    "log/syslog"
    "github.com/Sirupsen/logrus"
    "gopkg.in/gemnasium/logrus-airbrake-hook.v2" // the package is named "aibrake"
    )

func main() {
    log := logrus.New()
    log.AddHook(airbrake.NewHook(123, "xyz", "development"))
    log.Error("some logging message") // The error is sent to airbrake in background
}
```

Note that if environment == "development", the hook will not send anything to airbrake.


