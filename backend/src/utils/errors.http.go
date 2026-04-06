package utils

import (
	"errors"
	"net/http"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/jackc/pgx/v5/pgconn"
)

func WriteDomainError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, types.ErrInvalidName),
		errors.Is(err, types.ErrInvalidIdentifier),
		errors.Is(err, types.ErrInvalidEmail),
		errors.Is(err, types.ErrInvalidPassword),
		errors.Is(err, types.ErrInvalidVerificationToken),
		errors.Is(err, types.ErrInvalidProjectInvitation),
		errors.Is(err, types.ErrInvalidProjectID),
		errors.Is(err, types.ErrInvalidNodeID),
		errors.Is(err, types.ErrInvalidServiceID),
		errors.Is(err, types.ErrInvalidUserID),
		errors.Is(err, types.ErrNoFieldsToUpdate),
		errors.Is(err, types.ErrInvalidProjectMemberRole),
		errors.Is(err, types.ErrInvalidProjectInvitationRole),
		errors.Is(err, types.ErrEmptyProjectInvitations),
		errors.Is(err, types.ErrInvalidCheckType),
		errors.Is(err, types.ErrInvalidCheckTarget),
		errors.Is(err, types.ErrInvalidExecutionMode),
		errors.Is(err, types.ErrAgentExecutionRequiresNodeID),
		errors.Is(err, types.ErrManagedExecutionDisallowsNodeID),
		errors.Is(err, types.ErrNodeProjectMismatch),
		errors.Is(err, types.ErrNodeMustBeAgent),
		errors.Is(err, types.ErrAgentNodeMismatch),
		errors.Is(err, types.ErrServiceProjectMismatch),
		errors.Is(err, types.ErrServiceNodeMismatch),
		errors.Is(err, types.ErrCannotRemoveLastProjectOwner),
		errors.Is(err, types.ErrInvalidAlertRuleType),
		errors.Is(err, types.ErrUnsupportedAlertRuleType),
		errors.Is(err, types.ErrInvalidThresholdValue),
		errors.Is(err, types.ErrInvalidAlertStatus),
		errors.Is(err, types.ErrInvalidNotificationChannelType),
		errors.Is(err, types.ErrInvalidNotificationTarget),
		errors.Is(err, types.ErrInvalidNotificationDeliveryStatus):
		WriteError(w, http.StatusBadRequest, err.Error())
		return true
	case errors.Is(err, types.ErrUnauthenticated),
		errors.Is(err, types.ErrInvalidCredentials):
		WriteError(w, http.StatusUnauthorized, err.Error())
		return true
	case errors.Is(err, types.ErrForbidden),
		errors.Is(err, types.ErrProjectInvitationEmailMismatch):
		WriteError(w, http.StatusForbidden, err.Error())
		return true
	case errors.Is(err, types.ErrProjectNotFound),
		errors.Is(err, types.ErrNodeNotFound),
		errors.Is(err, types.ErrServiceNotFound),
		errors.Is(err, types.ErrSessionNotFound),
		errors.Is(err, types.ErrProjectMemberNotFound),
		errors.Is(err, types.ErrProjectInvitationNotFound),
		errors.Is(err, types.ErrAlertRuleNotFound),
		errors.Is(err, types.ErrNotificationRecipientNotFound),
		errors.Is(err, types.ErrUserNotFound):
		WriteError(w, http.StatusNotFound, err.Error())
		return true
	case errors.Is(err, types.ErrProjectMemberAlreadyExists):
		WriteError(w, http.StatusConflict, err.Error())
		return true
	case errors.Is(err, types.ErrNodeHasServices):
		WriteError(w, http.StatusConflict, err.Error())
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			WriteError(w, http.StatusConflict, "resource already exists")
			return true
		case "23503":
			WriteError(w, http.StatusBadRequest, "invalid related resource")
			return true
		}
	}

	return false
}
