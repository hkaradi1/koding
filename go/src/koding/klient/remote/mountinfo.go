package remote

import (
	"errors"
	"fmt"
	"koding/klient/remote/mount"
	"koding/klient/remote/req"

	"github.com/koding/kite"
)

// MountInfoHandler implements the Kite Handler for the remote.mountInfo method.
func (r *Remote) MountInfoHandler(kreq *kite.Request) (interface{}, error) {
	if kreq.Args == nil {
		return nil, errors.New("Required arguments were not passed.")
	}

	var params req.MountInfo
	if err := kreq.Args.One().Unmarshal(&params); err != nil {
		err = fmt.Errorf(
			"remote.mountInfo: Error '%s' while unmarshalling request '%s'\n",
			err, kreq.Args.One(),
		)
		r.log.Error("Error unmarshalling. err:%s", err)
		return nil, err
	}

	return r.MountInfo(params)
}

func (r *Remote) MountInfo(params req.MountInfo) (req.MountInfoResponse, error) {
	m, ok := r.mounts.FindByName(params.MountName)
	if !ok {
		r.log.Error(
			"MountInfo requested but mount could not be found. mount:%s",
			params.MountName,
		)
		return req.MountInfoResponse{}, mount.ErrMountNotFound
	}

	mountInfo := req.MountInfoResponse{
		MountFolder:      m.MountFolder,
		SyncIntervalOpts: m.SyncIntervalOpts,
	}

	return mountInfo, nil
}
