package services

import (
	"context"
	"fmt"

	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
)

type emailRuntimeConfigResolver interface {
	ResolveEmailConfig(ctx context.Context) (EmailRuntimeConfig, error)
}

type runtimeEmailSenderFactory func(config EmailRuntimeConfig) (emailpkg.Sender, error)

type RuntimeEmailProvider struct {
	configResolver emailRuntimeConfigResolver
	renderer       *emailpkg.Renderer
	senderFactory  runtimeEmailSenderFactory
}

func NewRuntimeEmailProvider(configResolver emailRuntimeConfigResolver) *RuntimeEmailProvider {
	return &RuntimeEmailProvider{
		configResolver: configResolver,
		renderer:       emailpkg.NewRenderer(),
		senderFactory:  newRuntimeEmailSender,
	}
}

func (p *RuntimeEmailProvider) Alerts(ctx context.Context) (*emailpkg.Service, error) {
	return p.serviceForPurpose(ctx, func(config EmailRuntimeConfig) string {
		return config.AlertsFrom
	})
}

func (p *RuntimeEmailProvider) Security(ctx context.Context) (*emailpkg.Service, error) {
	return p.serviceForPurpose(ctx, func(config EmailRuntimeConfig) string {
		return config.SecurityFrom
	})
}

func (p *RuntimeEmailProvider) Notifications(ctx context.Context) (*emailpkg.Service, error) {
	return p.serviceForPurpose(ctx, func(config EmailRuntimeConfig) string {
		return config.NotificationsFrom
	})
}

func (p *RuntimeEmailProvider) serviceForPurpose(ctx context.Context, fromAddress func(config EmailRuntimeConfig) string) (*emailpkg.Service, error) {
	if p == nil || p.configResolver == nil {
		return nil, nil
	}

	emailConfig, err := p.configResolver.ResolveEmailConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve email config: %w", err)
	}
	if !emailConfig.IsAvailable {
		return nil, nil
	}

	senderFactory := p.senderFactory
	if senderFactory == nil {
		senderFactory = newRuntimeEmailSender
	}

	sender, err := senderFactory(emailConfig)
	if err != nil {
		return nil, err
	}

	renderer := p.renderer
	if renderer == nil {
		renderer = emailpkg.NewRenderer()
	}

	return emailpkg.NewService(renderer, sender, fromAddress(emailConfig)), nil
}

func newRuntimeEmailSender(config EmailRuntimeConfig) (emailpkg.Sender, error) {
	switch config.Provider {
	case "resend":
		return emailpkg.NewResendSender(config.ResendAPIKey), nil
	default:
		return nil, fmt.Errorf("email provider %q is not supported", config.Provider)
	}
}
