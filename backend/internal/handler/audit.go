package handler

import (
	"time"

	"forward-panel/internal/model"
)

func auditLog(actorID uint, action string, targetID *uint, detail string) {
	model.DB.Create(&model.AuditLog{
		ActorID:   actorID,
		Action:    action,
		TargetID:  targetID,
		Detail:    detail,
		CreatedAt: time.Now(),
	})
}
