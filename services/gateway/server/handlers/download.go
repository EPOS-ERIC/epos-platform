package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) DownloadDistribution(ctx context.Context, params gatewayapi.DownloadDistributionParams) (*gatewayapi.DownloadDistribution2XXStatusCodeWithHeaders, error) {
	actorID, _ := authcontext.ActorID(ctx)

	download, err := h.downloads.Download(ctx, actorID, params.DistributionId)
	if err != nil {
		return nil, err
	}

	if download.Body == nil {
		return nil, apperror.Default(apperror.CodeInternal)
	}

	body := download.Body

	streaming := false
	defer func() {
		if !streaming {
			_ = body.Close()
		}
	}()

	status := download.StatusCode
	if status >= http.StatusBadRequest {
		payload, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		failure := apperror.Default(apperror.CodeUpstreamHTTPFailure)
		problemStatus, problem := problemResponse(ctx, failure)
		problem.UpstreamResponse = gatewayapi.NewOptProblemUpstreamResponse(gatewayapi.ProblemUpstreamResponse{
			Status: status,
			Body:   string(payload),
		})
		reportHTTPError(h.log, ctx, failure, problemStatus)

		return nil, &gatewayapi.ProblemStatusCode{StatusCode: problemStatus, Response: problem}
	}

	if status == http.StatusNoContent || status == http.StatusResetContent || status == http.StatusNotModified {
		if _, err := io.Copy(io.Discard, body); err != nil {
			return nil, err
		}

		return &gatewayapi.DownloadDistribution2XXStatusCodeWithHeaders{StatusCode: status}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	streaming = true

	return &gatewayapi.DownloadDistribution2XXStatusCodeWithHeaders{
		StatusCode:  status,
		ContentType: "application/octet-stream",
		Response:    gatewayapi.DownloadDistribution2XX{Data: body},
	}, nil
}
