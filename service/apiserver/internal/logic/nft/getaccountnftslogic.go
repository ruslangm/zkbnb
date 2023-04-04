package nft

import (
	"context"
	"github.com/bnb-chain/zkbnb/common"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
	types2 "github.com/bnb-chain/zkbnb/types"
)

const (
	queryByAccountIndex = "account_index"
	queryByL1Address    = "l1_address"
)

type GetAccountNftsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAccountNftsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountNftsLogic {
	return &GetAccountNftsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAccountNftsLogic) GetAccountNfts(req *types.ReqGetAccountNfts) (resp *types.Nfts, err error) {
	resp = &types.Nfts{
		Nfts: make([]*types.Nft, 0, int64(req.Offset)),
	}

	accountIndex := int64(0)
	switch req.By {
	case queryByAccountIndex:
		accountIndex, err = strconv.ParseInt(req.Value, 10, 64)
		if err != nil || accountIndex < 0 {
			return nil, types2.AppErrInvalidAccountIndex
		}
	case queryByL1Address:
		accountIndex, err = l.svcCtx.MemCache.GetAccountIndexByL1Address(req.Value)
	default:
		return nil, types2.AppErrInvalidParam.RefineError("param by should be account_index|l1_address")
	}

	if err != nil {
		if err == types2.DbErrNotFound {
			return resp, nil
		}
		return nil, types2.AppErrInternal
	}

	total, err := l.svcCtx.NftModel.GetNftsCountByAccountIndex(accountIndex)
	if err != nil {
		if err != types2.DbErrNotFound {
			return nil, types2.AppErrInternal
		}
	}

	resp.Total = total
	if total == 0 || total <= int64(req.Offset) {
		return resp, nil
	}

	nfts, err := l.svcCtx.NftModel.GetNftsByAccountIndex(accountIndex, int64(req.Limit), int64(req.Offset))
	if err != nil {
		return nil, types2.AppErrInternal
	}

	for _, nft := range nfts {
		creatorL1Address, _ := l.svcCtx.MemCache.GetL1AddressByIndex(nft.CreatorAccountIndex)
		ownerL1Address, _ := l.svcCtx.MemCache.GetL1AddressByIndex(nft.OwnerAccountIndex)
		resp.Nfts = append(resp.Nfts, &types.Nft{
			Index:               nft.NftIndex,
			CreatorAccountIndex: nft.CreatorAccountIndex,
			CreatorL1Address:    creatorL1Address,
			OwnerAccountIndex:   nft.OwnerAccountIndex,
			OwnerL1Address:      ownerL1Address,
			ContentHash:         nft.NftContentHash,
			RoyaltyRate:         nft.RoyaltyRate,
			CollectionId:        nft.CollectionId,
			IpfsId:              common.GenerateCid(nft.NftContentHash),
		})
	}
	return resp, nil
}
