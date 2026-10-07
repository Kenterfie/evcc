package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/gorilla/mux"
)

// minSocHandler updates min soc
func minSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		soc, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		v.SetMinSoc(soc)

		res := struct {
			Soc int `json:"soc"`
		}{
			Soc: v.GetMinSoc(),
		}

		jsonWrite(w, res)
	}
}

// limitSocHandler updates limit soc
func limitSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		soc, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		v.SetLimitSoc(soc)

		res := struct {
			Soc int `json:"soc"`
		}{
			Soc: v.GetLimitSoc(),
		}

		jsonWrite(w, res)
	}
}

// vehicleModeHandler updates the vehicle charge mode (empty value clears it)
func vehicleModeHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		mode, err := api.ChargeModeString(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		v.SetMode(mode)

		res := struct {
			Mode api.ChargeMode `json:"mode"`
		}{
			Mode: v.GetMode(),
		}

		jsonWrite(w, res)
	}
}

// vehicleAlwaysChargeHandler updates the vehicle always charge default (empty value clears it)
func vehicleAlwaysChargeHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// route restricts value to on|off, DELETE has none
		v.SetAlwaysCharge(api.AlwaysCharge(vars["value"]))

		res := struct {
			AlwaysCharge api.AlwaysCharge `json:"alwaysCharge"`
		}{
			AlwaysCharge: v.GetAlwaysCharge(),
		}

		jsonWrite(w, res)
	}
}

// planSocHandler updates plan soc and time
func planSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, err := time.ParseInLocation(time.RFC3339, vars["time"], nil)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		soc, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetPlanSoc(ts, soc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, soc = v.GetPlanSoc()

		res := struct {
			Soc  int       `json:"soc"`
			Time time.Time `json:"time"`
		}{
			Soc:  soc,
			Time: ts,
		}

		jsonWrite(w, res)
	}
}

func planStrategyHandlerSetter(r *http.Request, set func(api.PlanStrategy) error) error {
	var res api.PlanStrategy
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return err
	}

	return set(res)
}

// updatePlanStrategyHandler updates plan strategy
func updatePlanStrategyHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := planStrategyHandlerSetter(r, v.SetPlanStrategy); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := v.GetPlanStrategy()

		jsonWrite(w, res)
	}
}

// addRepeatingPlansHandler handles any information regarding weekday, hour, minute, soc and isActive
func addRepeatingPlansHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		var res []api.RepeatingPlan
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetRepeatingPlans(res); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, res)
	}
}

// planSocRemoveHandler removes plan soc and time
func planSocRemoveHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetPlanSoc(time.Time{}, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, struct{}{})
	}
}

// vehicleStatusHandler returns live vehicle status (soc, range, odometer,
// charge state). Values are served through the vehicle's cached getters so a
// call within the configured poll interval costs no additional API request.
func vehicleStatusHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		instance := v.Instance()

		res := struct {
			Soc          *float64         `json:"soc"`
			Range        *int64           `json:"range"`
			Odometer     *float64         `json:"odometer"`
			ChargeStatus api.ChargeStatus `json:"chargeStatus"`
			Updated      *time.Time       `json:"updated,omitempty"`
		}{
			ChargeStatus: api.StatusA,
		}

		// fall back to the scrape time when the data source provides no timestamp
		scrapeTime := time.Now()
		read := func(val any) {
			if val != nil && res.Updated == nil {
				res.Updated = &scrapeTime
			}
		}

		if b, ok := api.Cap[api.Battery](instance); ok {
			if soc, err := b.Soc(); err == nil && soc > 0 {
				res.Soc = &soc
				read(soc)
			}
		}

		if vr, ok := api.Cap[api.VehicleRange](instance); ok {
			if rng, err := vr.Range(); err == nil && rng > 0 {
				res.Range = &rng
				read(rng)
			}
		}

		if vo, ok := api.Cap[api.VehicleOdometer](instance); ok {
			if odo, err := vo.Odometer(); err == nil && odo > 0 {
				res.Odometer = &odo
				read(odo)
			}
		}

		if cs, ok := api.Cap[api.ChargeState](instance); ok {
			if status, err := cs.Status(); err == nil && status != api.StatusA {
				res.ChargeStatus = status
				read(status)
			}
		}

		// prefer the data source's own timestamp over evcc's scrape time
		if dt, ok := api.Cap[api.VehicleDataTimestamp](instance); ok {
			if ts, err := dt.DataUpdated(); err == nil && !ts.IsZero() {
				res.Updated = &ts
			}
		}

		jsonWrite(w, res)
	}
}
