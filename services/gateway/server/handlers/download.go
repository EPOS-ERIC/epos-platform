package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) DownloadDistribution(ctx context.Context, params gatewayapi.DownloadDistributionParams) (*gatewayapi.DownloadDistribution2XXStatusCode, error) {
	actorID, _ := authcontext.ActorID(ctx)

	download, err := h.distributionAccess.Download(ctx, actorID, params.DistributionId)
	if err != nil {
		return nil, err
	}

	download, err = h.prepareRetrievalResponse(ctx, download)
	if err != nil {
		return nil, err
	}

	response := &gatewayapi.DownloadDistribution2XXStatusCode{StatusCode: download.StatusCode}
	if download.Body != nil {
		response.Response.Data = download.Body
	}

	return response, nil
}

func (h *ogenHandler) prepareRetrievalResponse(ctx context.Context, download domain.RetrievalResponse) (domain.RetrievalResponse, error) {
	if download.Body == nil {
		return domain.RetrievalResponse{}, apperror.Default(apperror.CodeInternal)
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
			return domain.RetrievalResponse{}, err
		}

		if err := ctx.Err(); err != nil {
			return domain.RetrievalResponse{}, err
		}

		failure := apperror.Default(apperror.CodeUpstreamHTTPFailure)
		problemStatus, problem := problemResponse(ctx, failure)
		problem.UpstreamResponse = gatewayapi.NewOptProblemUpstreamResponse(gatewayapi.ProblemUpstreamResponse{
			Status: status,
			Body:   string(payload),
		})
		reportHTTPError(h.log, ctx, failure, problemStatus)

		return domain.RetrievalResponse{}, &gatewayapi.GatewayErrorStatusCode{StatusCode: problemStatus, Response: problem}
	}

	if status == http.StatusNoContent || status == http.StatusResetContent || status == http.StatusNotModified {
		if _, err := io.Copy(io.Discard, body); err != nil {
			return domain.RetrievalResponse{}, err
		}

		return domain.RetrievalResponse{StatusCode: status}, nil
	}

	if err := ctx.Err(); err != nil {
		return domain.RetrievalResponse{}, err
	}

	// TODO: Abort after committed headers on streaming failures instead of letting
	// ogen's error handler append a Problem. Shared recovery must preserve http.ErrAbortHandler.
	streaming = true

	return download, nil
}
