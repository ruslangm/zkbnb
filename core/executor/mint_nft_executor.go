package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bnb-chain/zkbnb-crypto/ffmath"
	"github.com/bnb-chain/zkbnb-crypto/wasm/txtypes"
	common2 "github.com/bnb-chain/zkbnb/common"
	nftModels "github.com/bnb-chain/zkbnb/core/model"
	"github.com/bnb-chain/zkbnb/dao/nft"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-openapi/swag"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"k8s.io/kube-openapi/pkg/validation/validate"
	"strings"
)

type MintNftExecutor struct {
	BaseExecutor

	txInfo *txtypes.MintNftTxInfo
}

func NewMintNftExecutor(bc IBlockchain, tx *tx.Tx) (TxExecutor, error) {
	txInfo, err := types.ParseMintNftTxInfo(tx.TxInfo)
	if err != nil {
		logx.Errorf("parse transfer tx failed: %s", err.Error())
		return nil, errors.New("invalid tx info")
	}

	return &MintNftExecutor{
		BaseExecutor: NewBaseExecutor(bc, tx, txInfo),
		txInfo:       txInfo,
	}, nil
}

func (e *MintNftExecutor) Prepare() error {
	txInfo := e.txInfo

	// Set the right nft index for tx info.
	nextNftIndex := e.bc.StateDB().GetNextNftIndex()
	txInfo.NftIndex = nextNftIndex

	if !e.bc.StateDB().DryRun {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		ids := id.String()
		ipnsName := fmt.Sprintf("%s-%d", ids, txInfo.NftIndex)
		ipnsId, err := common2.Ipfs.GenerateIPNS(ipnsName)
		if err != nil {
			return err
		}
		cid, err := sendToIpfs(&nftModels.NftMetaData{
			Image:       e.txInfo.MetaData.Image,
			Name:        e.txInfo.MetaData.Name,
			Description: e.txInfo.MetaData.Description,
			Attributes:  e.txInfo.MetaData.Attributes,
			Ipns:        fmt.Sprintf("%s%s", "https://ipfs.io/ipns/", ipnsId.Id),
		}, e.txInfo.NftIndex)
		if err != nil {
			return err
		}
		hash, err := common2.Ipfs.GenerateHash(cid)
		if err != nil {
			return err
		}
		txInfo.NftContentHash = hash
		txInfo.IpnsName = ipnsName
		txInfo.IpnsId = ipnsId.Id

	}
	// Mark the tree states that would be affected in this executor.
	e.MarkNftDirty(txInfo.NftIndex)
	e.MarkAccountAssetsDirty(txInfo.CreatorAccountIndex, []int64{txInfo.GasFeeAssetId})
	e.MarkAccountAssetsDirty(txInfo.GasAccountIndex, []int64{txInfo.GasFeeAssetId})
	e.MarkAccountAssetsDirty(txInfo.ToAccountIndex, []int64{})
	return e.BaseExecutor.Prepare()
}

func (e *MintNftExecutor) VerifyInputs(skipGasAmtChk, skipSigChk bool) error {
	txInfo := e.txInfo
	if err := e.Validate(); err != nil {
		return err
	}
	if txInfo.CreatorAccountIndex != txInfo.ToAccountIndex {
		return types.AppErrInvalidToAccount
	}
	err := e.BaseExecutor.VerifyInputs(skipGasAmtChk, skipSigChk)
	if err != nil {
		return err
	}

	creatorAccount, err := e.bc.StateDB().GetFormatAccount(txInfo.CreatorAccountIndex)
	if err != nil {
		return err
	}
	if creatorAccount.CollectionNonce <= txInfo.NftCollectionId {
		return types.AppErrInvalidCollectionId
	}
	if creatorAccount.AssetInfo[txInfo.GasFeeAssetId].Balance.Cmp(txInfo.GasFeeAssetAmount) < 0 {
		return types.AppErrBalanceNotEnough
	}

	toAccount, err := e.bc.StateDB().GetFormatAccount(txInfo.ToAccountIndex)
	if err != nil {
		return err
	}
	if txInfo.ToAccountNameHash != toAccount.AccountNameHash {
		return types.AppErrInvalidToAccountNameHash
	}
	return nil
}

func (e *MintNftExecutor) ApplyTransaction() error {
	bc := e.bc
	txInfo := e.txInfo

	// apply changes
	creatorAccount, err := bc.StateDB().GetFormatAccount(txInfo.CreatorAccountIndex)
	if err != nil {
		return err
	}

	creatorAccount.AssetInfo[txInfo.GasFeeAssetId].Balance = ffmath.Sub(creatorAccount.AssetInfo[txInfo.GasFeeAssetId].Balance, txInfo.GasFeeAssetAmount)
	creatorAccount.Nonce++

	stateCache := e.bc.StateDB()
	stateCache.SetPendingAccount(txInfo.CreatorAccountIndex, creatorAccount)

	bm, err := json.Marshal(txInfo.MetaData)
	if err != nil {
		return err
	}
	stateCache.SetPendingNft(txInfo.NftIndex, &nft.L2Nft{
		NftIndex:            txInfo.NftIndex,
		CreatorAccountIndex: txInfo.CreatorAccountIndex,
		OwnerAccountIndex:   txInfo.ToAccountIndex,
		NftContentHash:      txInfo.NftContentHash,
		CreatorTreasuryRate: txInfo.CreatorTreasuryRate,
		CollectionId:        txInfo.NftCollectionId,
		IpnsName:            txInfo.IpnsName,
		IpnsId:              txInfo.IpnsId,
		Metadata:            string(bm),
	})
	stateCache.SetPendingGas(txInfo.GasFeeAssetId, txInfo.GasFeeAssetAmount)
	return e.BaseExecutor.ApplyTransaction()
}

func sendToIpfs(data *nftModels.NftMetaData, nftIndex int64) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	cid, error := common2.Ipfs.Upload(b, nftIndex)

	if error != nil {
		return cid, error
	}
	return cid, nil
}

func (e *MintNftExecutor) GeneratePubData() error {
	txInfo := e.txInfo

	var buf bytes.Buffer
	buf.WriteByte(uint8(types.TxTypeMintNft))
	buf.Write(common2.Uint32ToBytes(uint32(txInfo.CreatorAccountIndex)))
	buf.Write(common2.Uint32ToBytes(uint32(txInfo.ToAccountIndex)))
	buf.Write(common2.Uint40ToBytes(txInfo.NftIndex))
	buf.Write(common2.Uint16ToBytes(uint16(txInfo.GasFeeAssetId)))
	packedFeeBytes, err := common2.FeeToPackedFeeBytes(txInfo.GasFeeAssetAmount)
	if err != nil {
		logx.Errorf("[ConvertTxToDepositPubData] unable to convert amount to packed fee amount: %s", err.Error())
		return err
	}
	buf.Write(packedFeeBytes)
	buf.Write(common2.Uint16ToBytes(uint16(txInfo.CreatorTreasuryRate)))
	buf.Write(common2.Uint16ToBytes(uint16(txInfo.NftCollectionId)))
	buf.Write(common2.PrefixPaddingBufToChunkSize(common.FromHex(txInfo.NftContentHash)))

	pubData := common2.SuffixPaddingBuToPubdataSize(buf.Bytes())

	stateCache := e.bc.StateDB()
	stateCache.PubData = append(stateCache.PubData, pubData...)
	return nil
}

func (e *MintNftExecutor) GetExecutedTx(fromApi bool) (*tx.Tx, error) {
	txInfoBytes, err := json.Marshal(e.txInfo)
	if err != nil {
		logx.Errorf("unable to marshal tx, err: %s", err.Error())
		return nil, errors.New("unmarshal tx failed")
	}

	e.tx.TxInfo = string(txInfoBytes)
	e.tx.GasFeeAssetId = e.txInfo.GasFeeAssetId
	e.tx.GasFee = e.txInfo.GasFeeAssetAmount.String()
	e.tx.NftIndex = e.txInfo.NftIndex
	return e.BaseExecutor.GetExecutedTx(fromApi)
}

func (e *MintNftExecutor) GenerateTxDetails() ([]*tx.TxDetail, error) {
	txInfo := e.txInfo

	copiedAccounts, err := e.bc.StateDB().DeepCopyAccounts([]int64{txInfo.CreatorAccountIndex, txInfo.ToAccountIndex, txInfo.GasAccountIndex})
	if err != nil {
		return nil, err
	}

	creatorAccount := copiedAccounts[txInfo.CreatorAccountIndex]
	toAccount := copiedAccounts[txInfo.ToAccountIndex]
	gasAccount := copiedAccounts[txInfo.GasAccountIndex]

	txDetails := make([]*tx.TxDetail, 0, 4)

	// from account gas asset
	order := int64(0)
	accountOrder := int64(0)
	txDetails = append(txDetails, &tx.TxDetail{
		AssetId:      txInfo.GasFeeAssetId,
		AssetType:    types.FungibleAssetType,
		AccountIndex: txInfo.CreatorAccountIndex,
		AccountName:  creatorAccount.AccountName,
		Balance:      creatorAccount.AssetInfo[txInfo.GasFeeAssetId].String(),
		BalanceDelta: types.ConstructAccountAsset(
			txInfo.GasFeeAssetId,
			ffmath.Neg(txInfo.GasFeeAssetAmount),
			types.ZeroBigInt,
		).String(),
		Order:           order,
		Nonce:           creatorAccount.Nonce,
		AccountOrder:    accountOrder,
		CollectionNonce: creatorAccount.CollectionNonce,
	})
	creatorAccount.AssetInfo[txInfo.GasFeeAssetId].Balance = ffmath.Sub(creatorAccount.AssetInfo[txInfo.GasFeeAssetId].Balance, txInfo.GasFeeAssetAmount)
	if creatorAccount.AssetInfo[txInfo.GasFeeAssetId].Balance.Cmp(types.ZeroBigInt) < 0 {
		return nil, errors.New("insufficient gas fee balance")
	}

	// to account empty delta
	order++
	accountOrder++
	txDetails = append(txDetails, &tx.TxDetail{
		AssetId:      txInfo.GasFeeAssetId,
		AssetType:    types.FungibleAssetType,
		AccountIndex: txInfo.ToAccountIndex,
		AccountName:  toAccount.AccountName,
		Balance:      toAccount.AssetInfo[txInfo.GasFeeAssetId].String(),
		BalanceDelta: types.ConstructAccountAsset(
			txInfo.GasFeeAssetId,
			types.ZeroBigInt,
			types.ZeroBigInt,
		).String(),
		Order:           order,
		Nonce:           toAccount.Nonce,
		AccountOrder:    accountOrder,
		CollectionNonce: toAccount.CollectionNonce,
	})

	// to account nft delta
	oldNftInfo := types.EmptyNftInfo(txInfo.NftIndex)
	newNftInfo := &types.NftInfo{
		NftIndex:            txInfo.NftIndex,
		CreatorAccountIndex: txInfo.CreatorAccountIndex,
		OwnerAccountIndex:   txInfo.ToAccountIndex,
		NftContentHash:      txInfo.NftContentHash,
		CreatorTreasuryRate: txInfo.CreatorTreasuryRate,
		CollectionId:        txInfo.NftCollectionId,
	}
	order++
	txDetails = append(txDetails, &tx.TxDetail{
		AssetId:         txInfo.NftIndex,
		AssetType:       types.NftAssetType,
		AccountIndex:    txInfo.ToAccountIndex,
		AccountName:     toAccount.AccountName,
		Balance:         oldNftInfo.String(),
		BalanceDelta:    newNftInfo.String(),
		Order:           order,
		Nonce:           toAccount.Nonce,
		AccountOrder:    types.NilAccountOrder,
		CollectionNonce: toAccount.CollectionNonce,
	})

	// gas account gas asset
	order++
	accountOrder++
	txDetails = append(txDetails, &tx.TxDetail{
		AssetId:      txInfo.GasFeeAssetId,
		AssetType:    types.FungibleAssetType,
		AccountIndex: txInfo.GasAccountIndex,
		AccountName:  gasAccount.AccountName,
		Balance:      gasAccount.AssetInfo[txInfo.GasFeeAssetId].String(),
		BalanceDelta: types.ConstructAccountAsset(
			txInfo.GasFeeAssetId,
			txInfo.GasFeeAssetAmount,
			types.ZeroBigInt,
		).String(),
		Order:           order,
		Nonce:           gasAccount.Nonce,
		AccountOrder:    accountOrder,
		CollectionNonce: gasAccount.CollectionNonce,
		IsGas:           true,
	})
	return txDetails, nil
}

func (e *MintNftExecutor) Validate() error {
	var res []error
	if err := e.validateName(); err != nil {
		res = append(res, err)
	}
	if err := e.validateImage(); err != nil {
		res = append(res, err)
	}
	if err := e.validateAttribute(); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		err := fmt.Sprintln(res)
		return errors.New(err)
	}
	return nil
}

func (e *MintNftExecutor) validateCollectionID() error {

	if err := validate.Required("collectionId", "body", e.txInfo.NftCollectionId); err != nil {
		return err
	}

	return nil
}

func (e *MintNftExecutor) validateCreatorEarningRate() error {

	if err := validate.Required("creatorEarningRate", "body", e.txInfo.CreatorTreasuryRate); err != nil {
		return err
	}

	return nil
}

func (e *MintNftExecutor) validateImage() error {

	if err := validate.Required("image", "body", e.txInfo.MetaData.Image); err != nil {
		return err
	}

	return nil
}

func (e *MintNftExecutor) validateName() error {

	if err := validate.Required("name", "body", e.txInfo.MetaData.Name); err != nil {
		return err
	}

	return nil
}

func (e *MintNftExecutor) validateAttribute() error {
	if swag.IsZero(e.txInfo.MetaData.Attributes) { // not required
		return nil
	}
	var res []error
	var result []*nftModels.AssetAttribute
	err := json.Unmarshal([]byte(e.txInfo.MetaData.Attributes), &result)
	if err != nil {
		return err
	}
	if swag.IsZero(result) { // not required
		return nil
	}
	for i := 0; i < len(result); i++ {
		if swag.IsZero(result[i]) { // not required
			continue
		}
		if result[i] != nil {
			if strings.ToLower(*result[i].Name) == "properties" {
				if err := result[i].ValidateValue(); err != nil {
					res = append(res, err)
				}
			} else {
				if err := result[i].Validate(); err != nil {
					res = append(res, err)
				}
			}
		}
	}
	if len(res) > 0 {
		err := fmt.Sprintln(res)
		return errors.New(err)
	}
	return nil
}
