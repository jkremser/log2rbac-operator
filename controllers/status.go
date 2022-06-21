package controllers

import (
	"context"
	"fmt"
	kremserv1 "github.com/jkremser/log2rbac-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	Requested  = 1
	InProgress = 2
	Synced     = 3
	NoChange   = 4
	Error      = 5
	Unknown    = 6
)

var lastTwo map[string][]byte

func UpdateStatus(c client.Client, ctx context.Context, res *kremserv1.RbacNegotiation, error bool, added bool) {
	if lastTwo == nil {
		lastTwo = make(map[string][]byte)
	}
	if lastTwo[res.Name] == nil {
		lastTwo[res.Name] = []byte{Unknown, Requested}
		res.Status.Status = "Requested"
		if added {
			res.Status.EntriesAdded = 1
		} else {
			res.Status.EntriesAdded = 0
		}
		updateTimeAndSave(c, ctx, res)
		return
	}
	if error {
		lastTwo[res.Name] = []byte{lastTwo[res.Name][1], Error}
		res.Status.Status = "Error"
	} else if added {
		res.Status.EntriesAdded += 1
		lastTwo[res.Name] = []byte{lastTwo[res.Name][1], InProgress}
		res.Status.Status = "InProgress"
	} else {
		if lastTwo[res.Name][0] == NoChange && lastTwo[res.Name][1] == NoChange {
			res.Status.Status = "Synced"
		} else {
			res.Status.Status = "InProgress"
		}
		lastTwo[res.Name] = []byte{lastTwo[res.Name][1], NoChange}
	}

	updateTimeAndSave(c, ctx, res)
}

func IsNotOlderThan(res *kremserv1.RbacNegotiation, seconds float64) bool {
	if res.Status.LastCheck.Time.IsZero() {
		return true
	}
	duration := time.Since(res.Status.LastCheck.Time)
	return duration.Seconds() < seconds
}

func updateTimeAndSave(c client.Client, ctx context.Context, res *kremserv1.RbacNegotiation) {
	res.Status.LastCheck = metav1.Now()
	if res.Status.EntriesAdded == 0 {
		res.Status.EntriesAdded = 0
	}
	if err := c.Status().Update(ctx, res); err != nil {
		fmt.Println(err)
	}
}
