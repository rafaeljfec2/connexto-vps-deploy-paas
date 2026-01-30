//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
)

func InitializeApplication() (*Application, func(), error) {
	wire.Build(AppSet)
	return nil, nil, nil
}
