package config

import (
	"github.com/short-loop/shortloop-go/common/models/data"
)

type UpdateListener interface {
	OnSuccessfulConfigUpdate(agentConfig data.AgentConfig)
	OnErroneousConfigUpdate()
}
