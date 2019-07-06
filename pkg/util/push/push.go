package push

import (
	"context"
	"net/http"

	"github.com/cortexproject/cortex/pkg/ingester/client"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/weaveworks/common/httpgrpc"
)

// Handler is a http.Handler which accepts WriteRequests.
func Handler(push func(context.Context, *client.WriteRequest) (*client.WriteResponse, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		compressionType := util.CompressionTypeFor(r.Header.Get("X-Prometheus-Remote-Write-Version"))
		var req client.PreallocWriteRequest
		req.Source = client.API
		_, err := util.ParseProtoReader(r.Context(), r.Body, &req, compressionType)
		logger := util.WithContext(r.Context(), util.Logger)
		if err != nil {
			level.Error(logger).Log("err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		/*
			if enableBilling {
				var samples int64
				for _, ts := range req.Timeseries {
					samples += int64(len(ts.Samples))
				}
				if err := d.emitBillingRecord(r.Context(), buf, samples); err != nil {
					level.Error(logger).Log("msg", "error emitting billing record", "err", err)
				}
			}
		*/

		if _, err := push(r.Context(), &req.WriteRequest); err != nil {
			resp, ok := httpgrpc.HTTPResponseFromError(err)
			if !ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if resp.GetCode() != 202 {
				level.Error(logger).Log("msg", "push error", "err", err)
			}
			http.Error(w, string(resp.Body), int(resp.Code))
		}
	})
}