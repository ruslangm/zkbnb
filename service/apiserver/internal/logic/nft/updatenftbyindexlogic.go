package nft

import (
	"context"
	common2 "github.com/bnb-chain/zkbnb/common"
	"github.com/bnb-chain/zkbnb/dao/nft"
	types2 "github.com/bnb-chain/zkbnb/types"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
)

type UpdateNftByIndexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateNftByIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateNftByIndexLogic {
	return &UpdateNftByIndexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateNftByIndexLogic) UpdateNftByIndex(req *types.ReqUpdateNft) (*types.History, error) {
	l2Nft, err := l.svcCtx.NftModel.GetNft(req.NftIndex)
	if err != nil {
		if err == types2.DbErrNotFound {
			return nil, types2.AppErrNftNotFound
		}
		return nil, types2.AppErrInternal
	}
	cid, err := common2.Ipfs.UploadIPNS(req.Mutable)
	if err != nil {
		return nil, err
	}
	_, err = common2.Ipfs.PublishWithDetails(cid, l2Nft.IpnsName)
	if err != nil {
		return nil, err
	}
	history := &nft.L2NftMetadataHistory{
		NftIndex: req.NftIndex,
		IpnsId:   l2Nft.IpnsId,
		Cid:      cid,
		Mutable:  req.Mutable,
	}
	err = l.svcCtx.NftMetadataHistoryModel.CreateL2NftMetadataHistoryInTransact(history)
	if err != nil {
		return nil, err
	}
	return &types.History{
		Mutable: req.Mutable,
	}, nil
}
