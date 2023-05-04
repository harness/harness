package cron

import (
	"github.com/google/wire"
	"github.com/harness/gitness/gitrpc/server"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideCronManager)

func ProvideCronManager(gitrpcconfig server.Config) *CronManager {
	cmngr := NewCronManager()
	_ = AddAllGitRPCCronJobs(cmngr, gitrpcconfig)
	return cmngr
}
