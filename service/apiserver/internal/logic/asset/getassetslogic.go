package asset

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
	types2 "github.com/bnb-chain/zkbnb/types"
)

type GetAssetsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAssetsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAssetsLogic {
	return &GetAssetsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAssetsLogic) GetAssets(req *types.ReqGetRange) (resp *types.Assets, err error) {
	total, err := l.svcCtx.MemCache.GetAssetTotalCountWithFallback(func() (interface{}, error) {
		return l.svcCtx.AssetModel.GetAssetsTotalCount()
	})
	if err != nil {
		return nil, types2.AppErrInternal
	}

	resp = &types.Assets{
		Assets: make([]*types.Asset, 0, req.Limit),
		Total:  uint32(total),
	}
	if total == 0 || total <= int64(req.Offset) {
		return resp, nil
	}

	assets, err := l.svcCtx.AssetModel.GetAssets(int64(req.Limit), int64(req.Offset))
	if err != nil {
		return nil, types2.AppErrInternal
	}

	resp.Assets = make([]*types.Asset, 0)
	for _, asset := range assets {
		assetPrice, err := l.svcCtx.PriceFetcher.GetCurrencyPrice(l.ctx, asset.AssetSymbol)
		if err != nil {
			return nil, types2.AppErrInternal
		}
		resp.Assets = append(resp.Assets, &types.Asset{
			Id:         asset.AssetId,
			Name:       asset.AssetName,
			Decimals:   asset.Decimals,
			Symbol:     asset.AssetSymbol,
			Address:    asset.L1Address,
			Price:      strconv.FormatFloat(assetPrice, 'E', -1, 64),
			IsGasAsset: asset.IsGasAsset,
			Icon:       fmt.Sprintf(iconBaseUrl, strings.ToLower(asset.AssetSymbol)),
		})
	}
	return resp, nil
}
