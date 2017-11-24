// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.
package controller

import (
	"github.com/Sirupsen/logrus"
	"net/http"
	"github.com/goodrain/rainbond/pkg/builder/model"
	"github.com/goodrain/rainbond/pkg/db"
	dbmodel "github.com/goodrain/rainbond/pkg/db/model"
	httputil "github.com/goodrain/rainbond/pkg/util/http"
	"github.com/go-chi/chi"
	"strings"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
)

func GetAppPublish(w http.ResponseWriter, r *http.Request) {
	serviceKey := strings.TrimSpace(chi.URLParam(r, "serviceKey"))
	appVersion := strings.TrimSpace(chi.URLParam(r, "appVersion"))
	appp,err:=db.GetManager().AppPublishDao().GetAppPublish(serviceKey,appVersion)
	if err != nil {
		httputil.ReturnError(r,w,404,err.Error())
	}
	httputil.ReturnSuccess(r, w, appp)
}
func GetVersionByEventID(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))

	version,err:=db.GetManager().VersionInfoDao().GetVersionByEventID(eventID)
	if err != nil {
		httputil.ReturnError(r,w,404,err.Error())
	}
	httputil.ReturnSuccess(r, w, version)
}
func GetVersionByServiceID(w http.ResponseWriter, r *http.Request) {
	serviceID := strings.TrimSpace(chi.URLParam(r, "serviceID"))

	versions,err:=db.GetManager().VersionInfoDao().GetVersionByServiceID(serviceID)
	if err != nil {
		httputil.ReturnError(r,w,404,err.Error())
	}
	httputil.ReturnSuccess(r, w, versions)
}
func UpdateDeliveredPath(w http.ResponseWriter, r *http.Request) {
	in,err:=ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	jsonc,err:=simplejson.NewJson(in)
	event,_:=jsonc.Get("event_id").String()
	dt,_:=jsonc.Get("type").String()
	dp,_:=jsonc.Get("path").String()

	version,err:=db.GetManager().VersionInfoDao().GetVersionByEventID(event)
	if err != nil {
		httputil.ReturnError(r,w,404,err.Error())
		return
	}

	version.DeliveredType=dt
	version.DeliveredPath=dp
	err=db.GetManager().VersionInfoDao().UpdateModel(version)
	if err != nil {
		httputil.ReturnError(r,w,500,err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
	return
}




func AddAppPublish(w http.ResponseWriter, r *http.Request) {
	result := new(model.AppPublish)
	b,_:=ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	j,err:=simplejson.NewJson(b)
	if err != nil {
		logrus.Errorf("error decode json,details %s",err.Error())
		httputil.ReturnError(r,w,400,"bad request")
		return
	}

	result.AppVersion,err=j.Get("app_version").String()

	result.ServiceKey,err=j.Get("service_key").String()
	result.Slug,err=j.Get("slug").String()
	if err != nil {
		logrus.Warnf("error get slag json,details %s",err.Error())
	}
	result.Image,err=j.Get("image").String()
	if err != nil {
		logrus.Warnf("error get slag json,details %s",err.Error())
	}
	result.DestYS,err=j.Get("dest_ys").Bool()
	if err != nil {
		logrus.Warnf("error get slag json,details %s",err.Error())
	}
	result.DestYB,err=j.Get("dest_yb").Bool()
	if err != nil {
		logrus.Warnf("error get slag json,details %s",err.Error())
	}
	result.ShareID,err=j.Get("share_id").String()
	if err != nil {
		logrus.Warnf("error get slag json,details %s",err.Error())
	}

	dbmodel:=convertPublishToDB(result)
	//checkAndGet
	db.GetManager().AppPublishDao().AddModel(dbmodel)
	httputil.ReturnSuccess(r, w, nil)
}
func convertPublishToDB(publish *model.AppPublish) *dbmodel.AppPublish {

	dbm:=dbmodel.AppPublish{}
	dbm.ShareID=publish.ShareID
	dbm.AppVersion=publish.AppVersion
	dbm.DestYB=publish.DestYB
	dbm.DestYS=publish.DestYS
	dbm.Image=publish.Image
	dbm.ServiceKey=publish.ServiceKey
	dbm.Slug=publish.Slug
	return &dbm
}
