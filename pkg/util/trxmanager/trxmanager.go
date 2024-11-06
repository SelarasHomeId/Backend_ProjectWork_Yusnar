package trxmanager

import (
	"errors"
	"fmt"
	"runtime/debug"
	"selarashomeid/internal/abstraction"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type trxManager struct {
	db *gorm.DB
}

type trxFn func(ctx *abstraction.Context) error

func New(db *gorm.DB) *trxManager {
	return &trxManager{db}
}

func (g *trxManager) WithTrx(pCtx *abstraction.Context, fn trxFn) (err error) {
	tx := g.db.Begin()
	pCtx.Trx = &abstraction.TrxContext{
		Db: tx,
	}

	defer func() {
		if p := recover(); p != nil {
			// a panic occurred, rollback and repanic
			tx.Rollback()
			stackTrace := string(debug.Stack())
			logrus.Errorf("panic happened because: %v\nStack Trace:\n%s", p, stackTrace)
			err = errors.New("panic happened because: " + fmt.Sprintf("%v", p))
		} else if err != nil {
			// error occurred, rollback
			tx.Rollback()
		} else {
			// all good, commit
			err = tx.Commit().Error
		}
	}()

	err = fn(pCtx)
	return err
}
