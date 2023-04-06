package signature

import (
	"fmt"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/logic/utils"
	"github.com/bnb-chain/zkbnb/types"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

const (

	// SignatureTemplateWithdrawal Withdrawal ${amount} to: ${to.toLowerCase()}\nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce}
	SignatureTemplateWithdrawal = "Withdrawal %s to: %s\nFee: %s %d\nNonce: %d"
	// SignatureTemplateTransfer /* Transfer ${amount} ${tokenAddress} to: ${to.toLowerCase()}\nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce} */
	SignatureTemplateTransfer = "Transfer %s %d to: %d\nFee: %s %d\nNonce: %d"
	// SignatureTemplateCreateCollection CreateCollection ${accountIndex} ${collectionName} \nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce}
	SignatureTemplateCreateCollection = "CreateCollection %d %s \nFee: %s %d\nNonce: %d"
	// SignatureTemplateMintNft MintNFT ${contentHash} for: ${recipient.toLowerCase()}\nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce}
	SignatureTemplateMintNft = "MintNFT %s for: %d\nFee: %s %d\nNonce: %d"
	// SignatureTemplateTransferNft TransferNFT ${NftIndex} ${fromAccountIndex} to ${toAccountIndex} \nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce}
	SignatureTemplateTransferNft = "TransferNFT %d %d to %d \nFee: %s %d\nNonce: %d"
	// SignatureTemplateWithdrawalNft Withdrawal ${tokenIndex} to: ${to.toLowerCase()}\nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce}
	SignatureTemplateWithdrawalNft = "Withdrawal %d to: %s\nFee: %s %d\nNonce: %d"
	// SignatureTemplateCancelOffer CancelOffer ${offerId} by: ${accountIndex} \nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce}
	SignatureTemplateCancelOffer = "CancelOffer %d by: %d \nFee: %s %d\nNonce: %d"
	// SignatureTemplateAtomicMatch AtomicMatch ${amount} ${offerId} ${nftIndex} ${accountIndex} \nFee: ${fee} ${feeTokenAddress}\nNonce: ${nonce}
	SignatureTemplateAtomicMatch = "AtomicMatch %s %d %d %d \nFee: %s %d\nNonce: %d"

	// SignatureTemplateAccount AccountIndex:{AccountIndex}\nNftIndex:{NftIndex}\nNonce:{Nonce}
	SignatureTemplateAccount = "AccountIndex:%d\nNftIndex:%d\nNonce:%d"
)

var SignatureFunctionMap = make(map[uint32]func(txInfo string) (string, error), 0)

func GenerateSignatureBody(txType uint32, txInfo string) (string, error) {
	if len(SignatureFunctionMap) == 0 {
		ConstructSignatureFunction()
	}

	SignatureFunc := SignatureFunctionMap[txType]
	if SignatureFunc == nil {
		return "", errors.New(fmt.Sprintf("Can not find Signature Function for TxType:%d", txType))
	}

	signatureBody, err := SignatureFunc(txInfo)
	if err != nil {
		return "", err
	}
	return signatureBody, nil
}

func ConstructSignatureFunction() {
	SignatureFunctionMap[types.TxTypeWithdraw] = SignatureForWithdrawal
	SignatureFunctionMap[types.TxTypeTransfer] = SignatureForTransfer
	SignatureFunctionMap[types.TxTypeCreateCollection] = SignatureForCreateCollection
	SignatureFunctionMap[types.TxTypeMintNft] = SignatureForMintNft
	SignatureFunctionMap[types.TxTypeTransferNft] = SignatureForTransferNft
	SignatureFunctionMap[types.TxTypeWithdrawNft] = SignatureForWithdrawalNft
	SignatureFunctionMap[types.TxTypeCancelOffer] = SignatureForCancelOffer
	SignatureFunctionMap[types.TxTypeAtomicMatch] = SignatureForAtomicMatch
	SignatureFunctionMap[types.TxTypeEmpty] = SignatureForAccount

}

func SignatureForWithdrawal(txInfo string) (string, error) {
	transaction, err := types.ParseWithdrawTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse withdrawal tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateWithdrawal, utils.FormatWeiToEtherStr(transaction.AssetAmount), transaction.ToAddress,
		utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForTransfer(txInfo string) (string, error) {
	transaction, err := types.ParseTransferTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse transfer tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateTransfer, utils.FormatWeiToEtherStr(transaction.AssetAmount), transaction.FromAccountIndex,
		transaction.ToAccountIndex, utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForCreateCollection(txInfo string) (string, error) {
	transaction, err := types.ParseCreateCollectionTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse create collection tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateCreateCollection, transaction.AccountIndex,
		transaction.Name, utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForMintNft(txInfo string) (string, error) {
	transaction, err := types.ParseMintNftTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse mint nft tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateMintNft, transaction.ToAccountNameHash,
		transaction.ToAccountIndex, utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForTransferNft(txInfo string) (string, error) {
	transaction, err := types.ParseTransferNftTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse cancel offer tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateTransferNft, transaction.NftIndex, transaction.FromAccountIndex,
		transaction.ToAccountIndex, utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForWithdrawalNft(txInfo string) (string, error) {
	transaction, err := types.ParseWithdrawNftTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse withdrawal nft tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateWithdrawalNft, transaction.NftIndex,
		transaction.ToAddress, utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForCancelOffer(txInfo string) (string, error) {
	transaction, err := types.ParseCancelOfferTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse cancel offer tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateCancelOffer, transaction.OfferId,
		transaction.AccountIndex, utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForAtomicMatch(txInfo string) (string, error) {
	transaction, err := types.ParseAtomicMatchTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse atomic match tx failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}

	offer := transaction.BuyOffer
	if offer == nil {
		offer = transaction.SellOffer
	}
	if offer == nil {
		return "", errors.New("both buyOffer and sellOffer does not exist")
	}

	signatureBody := fmt.Sprintf(SignatureTemplateAtomicMatch, utils.FormatWeiToEtherStr(offer.AssetAmount), offer.OfferId, offer.NftIndex,
		transaction.AccountIndex, utils.FormatWeiToEtherStr(transaction.GasFeeAssetAmount), transaction.GasAccountIndex, transaction.Nonce)
	return signatureBody, nil
}

func SignatureForAccount(txInfo string) (string, error) {
	transaction, err := types.ParseUpdateNftTxInfo(txInfo)
	if err != nil {
		logx.Errorf("parse atomic match nft info failed: %s", err.Error())
		return "", errors.New("invalid tx info")
	}
	signatureBody := fmt.Sprintf(SignatureTemplateAccount, transaction.AccountIndex, transaction.NftIndex, transaction.Nonce)
	return signatureBody, nil
}
