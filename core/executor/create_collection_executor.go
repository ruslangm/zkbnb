package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-openapi/swag"
	"k8s.io/kube-openapi/pkg/validation/validate"
	"strconv"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb-crypto/ffmath"
	"github.com/bnb-chain/zkbnb-crypto/wasm/txtypes"
	common2 "github.com/bnb-chain/zkbnb/common"
	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/types"
)

type CreateCollectionExecutor struct {
	BaseExecutor

	txInfo *txtypes.CreateCollectionTxInfo
}

func NewCreateCollectionExecutor(bc IBlockchain, tx *tx.Tx) (TxExecutor, error) {
	txInfo, err := types.ParseCreateCollectionTxInfo(tx.TxInfo)
	if err != nil {
		logx.Errorf("parse transfer tx failed: %s", err.Error())
		return nil, errors.New("invalid tx info")
	}

	return &CreateCollectionExecutor{
		BaseExecutor: NewBaseExecutor(bc, tx, txInfo),
		txInfo:       txInfo,
	}, nil
}

func (e *CreateCollectionExecutor) Prepare() error {
	txInfo := e.txInfo

	// Mark the tree states that would be affected in this executor.
	e.MarkAccountAssetsDirty(txInfo.AccountIndex, []int64{txInfo.GasFeeAssetId})
	e.MarkAccountAssetsDirty(txInfo.GasAccountIndex, []int64{txInfo.GasFeeAssetId})
	err := e.BaseExecutor.Prepare()
	if err != nil {
		return err
	}

	// Set the right collection nonce to tx info.
	account, err := e.bc.StateDB().GetFormatAccount(txInfo.AccountIndex)
	if err != nil {
		return err
	}
	txInfo.CollectionId = account.CollectionNonce
	return nil
}

func (e *CreateCollectionExecutor) VerifyInputs(skipGasAmtChk, skipSigChk bool) error {
	txInfo := e.txInfo
	if err := e.Validate(); err != nil {
		return err
	}

	err := e.BaseExecutor.VerifyInputs(skipGasAmtChk, skipSigChk)
	if err != nil {
		return err
	}

	fromAccount, err := e.bc.StateDB().GetFormatAccount(txInfo.AccountIndex)
	if err != nil {
		return err
	}
	if fromAccount.AssetInfo[txInfo.GasFeeAssetId].Balance.Cmp(txInfo.GasFeeAssetAmount) < 0 {
		return types.AppErrBalanceNotEnough
	}

	return nil
}

func (e *CreateCollectionExecutor) ApplyTransaction() error {
	bc := e.bc
	txInfo := e.txInfo

	fromAccount, err := bc.StateDB().GetFormatAccount(txInfo.AccountIndex)
	if err != nil {
		return err
	}

	// apply changes
	fromAccount.AssetInfo[txInfo.GasFeeAssetId].Balance = ffmath.Sub(fromAccount.AssetInfo[txInfo.GasFeeAssetId].Balance, txInfo.GasFeeAssetAmount)
	fromAccount.Nonce++
	fromAccount.CollectionNonce++

	stateCache := e.bc.StateDB()
	stateCache.SetPendingAccount(fromAccount.AccountIndex, fromAccount)
	stateCache.SetPendingGas(txInfo.GasFeeAssetId, txInfo.GasFeeAssetAmount)
	return e.BaseExecutor.ApplyTransaction()
}

func (e *CreateCollectionExecutor) GeneratePubData() error {
	txInfo := e.txInfo

	var buf bytes.Buffer
	buf.WriteByte(uint8(types.TxTypeCreateCollection))
	buf.Write(common2.Uint32ToBytes(uint32(txInfo.AccountIndex)))
	buf.Write(common2.Uint16ToBytes(uint16(txInfo.CollectionId)))
	buf.Write(common2.Uint16ToBytes(uint16(txInfo.GasFeeAssetId)))
	packedFeeBytes, err := common2.FeeToPackedFeeBytes(txInfo.GasFeeAssetAmount)
	if err != nil {
		logx.Errorf("unable to convert amount to packed fee amount: %s", err.Error())
		return err
	}
	buf.Write(packedFeeBytes)

	pubData := common2.SuffixPaddingBuToPubdataSize(buf.Bytes())
	stateCache := e.bc.StateDB()
	stateCache.PubData = append(stateCache.PubData, pubData...)
	return nil
}

func (e *CreateCollectionExecutor) GetExecutedTx(fromApi bool) (*tx.Tx, error) {
	txInfoBytes, err := json.Marshal(e.txInfo)
	if err != nil {
		logx.Errorf("unable to marshal tx, err: %s", err.Error())
		return nil, errors.New("unmarshal tx failed")
	}

	e.tx.TxInfo = string(txInfoBytes)
	e.tx.GasFeeAssetId = e.txInfo.GasFeeAssetId
	e.tx.GasFee = e.txInfo.GasFeeAssetAmount.String()
	e.tx.CollectionId = e.txInfo.CollectionId
	return e.BaseExecutor.GetExecutedTx(fromApi)
}

func (e *CreateCollectionExecutor) GenerateTxDetails() ([]*tx.TxDetail, error) {
	txInfo := e.txInfo

	copiedAccounts, err := e.bc.StateDB().DeepCopyAccounts([]int64{txInfo.AccountIndex, txInfo.GasAccountIndex})
	if err != nil {
		return nil, err
	}

	fromAccount := copiedAccounts[txInfo.AccountIndex]
	gasAccount := copiedAccounts[txInfo.GasAccountIndex]

	txDetails := make([]*tx.TxDetail, 0, 4)

	// from account collection nonce
	order := int64(0)
	accountOrder := int64(0)
	txDetails = append(txDetails, &tx.TxDetail{
		AssetId:         types.NilAssetId,
		AssetType:       types.CollectionNonceAssetType,
		AccountIndex:    txInfo.AccountIndex,
		AccountName:     fromAccount.AccountName,
		Balance:         strconv.FormatInt(fromAccount.CollectionNonce, 10),
		BalanceDelta:    strconv.FormatInt(fromAccount.CollectionNonce+1, 10),
		Order:           order,
		Nonce:           fromAccount.Nonce,
		AccountOrder:    accountOrder,
		CollectionNonce: fromAccount.CollectionNonce,
	})
	fromAccount.CollectionNonce = fromAccount.CollectionNonce + 1

	// from account gas
	order++
	txDetails = append(txDetails, &tx.TxDetail{
		AssetId:      txInfo.GasFeeAssetId,
		AssetType:    types.FungibleAssetType,
		AccountIndex: txInfo.AccountIndex,
		AccountName:  fromAccount.AccountName,
		Balance:      fromAccount.AssetInfo[txInfo.GasFeeAssetId].String(),
		BalanceDelta: types.ConstructAccountAsset(
			txInfo.GasFeeAssetId,
			ffmath.Neg(txInfo.GasFeeAssetAmount),
			types.ZeroBigInt,
		).String(),
		Order:           order,
		Nonce:           fromAccount.Nonce,
		AccountOrder:    accountOrder,
		CollectionNonce: fromAccount.CollectionNonce,
	})
	fromAccount.AssetInfo[txInfo.GasFeeAssetId].Balance = ffmath.Sub(fromAccount.AssetInfo[txInfo.GasFeeAssetId].Balance, txInfo.GasFeeAssetAmount)
	if fromAccount.AssetInfo[txInfo.GasFeeAssetId].Balance.Cmp(types.ZeroBigInt) < 0 {
		return nil, errors.New("insufficient gas fee balance")
	}

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
		Order:        order,
		Nonce:        gasAccount.Nonce,
		AccountOrder: accountOrder,
		IsGas:        true,
	})
	return txDetails, nil
}

func (e *CreateCollectionExecutor) Validate() error {
	if err := validate.Required("MetaData", "body", e.txInfo.MetaData); err != nil {
		return err
	}

	var res []error
	// Mandatory
	if err := e.validateCategoryID(); err != nil {
		res = append(res, err)
	}
	if err := e.validateShortname(); err != nil {
		res = append(res, err)
	}

	// Not mandatory
	if err := e.validateBannerImage(); err != nil {
		res = append(res, err)
	}

	if err := e.validateDiscordLink(); err != nil {
		res = append(res, err)
	}

	if err := e.validateExternalLink(); err != nil {
		res = append(res, err)
	}

	if err := e.validateFeaturedImage(); err != nil {
		res = append(res, err)
	}

	if err := e.validateInstagramUserName(); err != nil {
		res = append(res, err)
	}

	if err := e.validateLogoImage(); err != nil {
		res = append(res, err)
	}

	if err := e.validateShortname(); err != nil {
		res = append(res, err)
	}

	if err := e.validateTelegramLink(); err != nil {
		res = append(res, err)
	}

	if err := e.validateTwitterUserName(); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		err := fmt.Sprintln(res)
		return errors.New(err)
	}
	return nil
}

func (e *CreateCollectionExecutor) validateBannerImage() error {
	if swag.IsZero(e.txInfo.MetaData.BannerImage) { // not required
		return nil
	}

	if err := validate.MinLength("bannerImage", "body", e.txInfo.MetaData.BannerImage, 4); err != nil {
		return err
	}

	if err := validate.MaxLength("bannerImage", "body", e.txInfo.MetaData.BannerImage, 256); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateCategoryID() error {

	if err := validate.Required("categoryId", "body", e.txInfo.MetaData.CategoryID); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateDiscordLink() error {
	if swag.IsZero(e.txInfo.MetaData.DiscordLink) { // not required
		return nil
	}

	if err := validate.MinLength("discordLink", "body", e.txInfo.MetaData.DiscordLink, 3); err != nil {
		return err
	}

	if err := validate.MaxLength("discordLink", "body", e.txInfo.MetaData.DiscordLink, 64); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateExternalLink() error {
	if swag.IsZero(e.txInfo.MetaData.ExternalLink) { // not required
		return nil
	}

	if err := validate.MinLength("externalLink", "body", e.txInfo.MetaData.ExternalLink, 4); err != nil {
		return err
	}

	if err := validate.MaxLength("externalLink", "body", e.txInfo.MetaData.ExternalLink, 64); err != nil {
		return err
	}

	if err := validate.Pattern("externalLink", "body", e.txInfo.MetaData.ExternalLink, `^[a-zA-z]+://[^\s]*$`); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateFeaturedImage() error {
	if swag.IsZero(e.txInfo.MetaData.FeaturedImage) { // not required
		return nil
	}

	if err := validate.MinLength("featuredImage", "body", e.txInfo.MetaData.FeaturedImage, 4); err != nil {
		return err
	}

	if err := validate.MaxLength("featuredImage", "body", e.txInfo.MetaData.FeaturedImage, 256); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateInstagramUserName() error {
	if swag.IsZero(e.txInfo.MetaData.InstagramUserName) { // not required
		return nil
	}

	if err := validate.MinLength("instagramUserName", "body", e.txInfo.MetaData.InstagramUserName, 3); err != nil {
		return err
	}

	if err := validate.MaxLength("instagramUserName", "body", e.txInfo.MetaData.InstagramUserName, 64); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateLogoImage() error {
	if swag.IsZero(e.txInfo.MetaData.LogoImage) { // not required
		return nil
	}

	if err := validate.MinLength("logoImage", "body", e.txInfo.MetaData.LogoImage, 4); err != nil {
		return err
	}

	if err := validate.MaxLength("logoImage", "body", e.txInfo.MetaData.LogoImage, 256); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateShortname() error {

	if err := validate.Required("shortname", "body", e.txInfo.MetaData.Shortname); err != nil {
		return err
	}

	if err := validate.MinLength("shortname", "body", e.txInfo.MetaData.Shortname, 3); err != nil {
		return err
	}

	if err := validate.MaxLength("shortname", "body", e.txInfo.MetaData.Shortname, 64); err != nil {
		return err
	}

	if err := validate.Pattern("shortname", "body", e.txInfo.MetaData.Shortname, `^\d*[a-zA-Z_][a-zA-Z0-9_]*$`); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateTelegramLink() error {
	if swag.IsZero(e.txInfo.MetaData.TelegramLink) { // not required
		return nil
	}

	if err := validate.MinLength("telegramLink", "body", e.txInfo.MetaData.TelegramLink, 3); err != nil {
		return err
	}

	if err := validate.MaxLength("telegramLink", "body", e.txInfo.MetaData.TelegramLink, 64); err != nil {
		return err
	}

	return nil
}

func (e *CreateCollectionExecutor) validateTwitterUserName() error {
	if swag.IsZero(e.txInfo.MetaData.TwitterUserName) { // not required
		return nil
	}

	if err := validate.MinLength("twitterUserName", "body", e.txInfo.MetaData.TwitterUserName, 3); err != nil {
		return err
	}

	if err := validate.MaxLength("twitterUserName", "body", e.txInfo.MetaData.TwitterUserName, 64); err != nil {
		return err
	}

	return nil
}
