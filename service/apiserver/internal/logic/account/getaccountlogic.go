package account

import (
	"context"
	"math/big"
	"sort"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
	types2 "github.com/bnb-chain/zkbnb/types"
)

const (
	queryByIndex     = "index"
	queryByL1Address = "l1_address"
)

type GetAccountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountLogic {
	return &GetAccountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAccountLogic) GetAccount(req *types.ReqGetAccount) (resp *types.Account, err error) {
	index := int64(-1)
	switch req.By {
	case queryByIndex:
		index, err = strconv.ParseInt(req.Value, 10, 64)
		if err != nil || index < 0 {
			return nil, types2.AppErrInvalidAccountIndex
		}
	case queryByL1Address:
		index, err = l.svcCtx.MemCache.GetAccountIndexByL1Address(req.Value)
		if err != nil {
			if err == types2.DbErrNotFound {
				return nil, types2.AppErrAccountNotFound
			}
			return nil, types2.AppErrInternal
		}
	default:
		return nil, types2.AppErrInvalidParam.RefineError("param by should be index|l1address")
	}

	if index < 0 {
		return nil, types2.AppErrAccountNotFound
	}
	account, err := l.svcCtx.StateFetcher.GetLatestAccount(index)
	if err != nil {
		if err == types2.DbErrNotFound {
			return nil, types2.AppErrAccountNotFound
		}
		return nil, types2.AppErrInternal
	}

	//todo  need to confirm whether there is any impact on the function
	//maxAssetId, err := l.svcCtx.AssetModel.GetMaxAssetId()
	//if err != nil {
	//	return nil, types2.AppErrInternal
	//}

	resp = &types.Account{
		Index:     account.AccountIndex,
		Status:    uint32(account.Status),
		L1Address: account.L1Address,
		Pk:        account.PublicKey,
		Nonce:     account.Nonce,
		Assets:    make([]*types.AccountAsset, 0, len(account.AssetInfo)),
	}

	totalAssetValue := big.NewFloat(0)

	for _, asset := range account.AssetInfo {
		//todo  need to confirm whether there is any impact on the function
		//if asset.AssetId > maxAssetId {
		//	continue //it is used for offer related, or empty balance; max ip id should be less than max asset id
		//}
		if asset.Balance == nil || asset.Balance.Cmp(types2.ZeroBigInt) == 0 {
			continue
		}
		if asset.Balance != nil && asset.Balance.Cmp(types2.ZeroBigInt) > 0 {
			var assetName, assetSymbol string
			var assetPrice float64

			assetInfo, err := l.svcCtx.MemCache.GetAssetByIdWithFallback(asset.AssetId, func() (interface{}, error) {

				return l.svcCtx.AssetModel.GetAssetById(asset.AssetId)
			})
			if err != nil {
				return nil, types2.AppErrInternal
			}
			assetName = assetInfo.AssetName
			assetSymbol = assetInfo.AssetSymbol

			assetPrice, err = l.svcCtx.PriceFetcher.GetCurrencyPrice(l.ctx, assetSymbol)
			if err != nil {
				return nil, types2.AppErrInternal
			}
			resp.Assets = append(resp.Assets, &types.AccountAsset{
				Id:      uint32(asset.AssetId),
				Name:    assetName,
				Balance: asset.Balance.String(),
				Price:   strconv.FormatFloat(assetPrice, 'E', -1, 64),
			})

			// BNB for example:
			//   1. Convert unit of balance from wei to BNB
			//   2. Calculate the result of (BNB balance * price per BNB)
			balanceInFloat := new(big.Float).SetInt(asset.Balance)
			unitConversion := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(assetInfo.Decimals)), nil)
			assetValue := balanceInFloat.Mul(
				new(big.Float).Quo(balanceInFloat, new(big.Float).SetInt(unitConversion)),
				big.NewFloat(assetPrice),
			)

			totalAssetValue = totalAssetValue.Add(totalAssetValue, assetValue)
		}
	}

	resp.TotalAssetValue = totalAssetValue.Text('f', -1)

	sort.Slice(resp.Assets, func(i, j int) bool {
		return resp.Assets[i].Id < resp.Assets[j].Id
	})

	return resp, nil
}
