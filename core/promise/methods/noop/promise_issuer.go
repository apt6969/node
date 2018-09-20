/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package noop

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/promise"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
)

const issuerLogPrefix = "[promise-issuer] "

// NewPromiseIssuer creates instance of PromiseIssuer
func NewPromiseIssuer(dialog communication.Dialog) *PromiseIssuer {
	return &PromiseIssuer{
		dialog: dialog,
	}
}

// PromiseIssuer issues promises in such way, what no actual money is added to promise
type PromiseIssuer struct {
	dialog communication.Dialog

	// these are populated by Start at runtime
	proposal dto.ServiceProposal
}

// Start issuing promises for given service proposal
func (issuer *PromiseIssuer) Start(proposal dto.ServiceProposal) error {
	issuer.proposal = proposal

	if _, err := issuer.sendNewPromise(); err != nil {
		// TODO Handle response for send promise
		return err
	}

	return issuer.subscribePromiseBalance()
}

// Stop stops issuing promises
func (issuer *PromiseIssuer) Stop() error {
	// TODO Should unregister consumers(subscriptions) here
	return nil
}

func (issuer *PromiseIssuer) sendNewPromise() (*promise.Response, error) {
	unsignedPromise := promise.NewPromise(issuer.IssuerID, identity.FromAddress(issuer.proposal.ProviderID), money.NewMoney(10, "MYST"))
	signedPromise, err := promise.SignByIssuer(unsignedPromise, issuer.Signer)
	if err != nil {
		return nil, err
	}

	return promise.Send(signedPromise, issuer.Dialog)
}

func (issuer *PromiseIssuer) subscribePromiseBalance() error {
	return issuer.dialog.Receive(
		&promise.BalanceMessageConsumer{issuer.processBalanceMessage},
	)
}

func (issuer *PromiseIssuer) processBalanceMessage(message promise.BalanceMessage) error {
	if !message.Accepted {
		log.Warn(issuerLogPrefix, fmt.Sprintf("Promise balance rejected: %s", message.Balance.String()))
	}

	log.Info(issuerLogPrefix, fmt.Sprintf("Promise balance notified: %s", message.Balance.String()))
	return nil
}

func (issuer *PromiseIssuer) subscribePromiseBalance() error {
	subscribeError := issuer.Dialog.Receive(
		&promise.BalanceMessageConsumer{issuer.processBalanceMessage},
	)
	if subscribeError != nil {
		return subscribeError
	}

	return nil
}

func (issuer *PromiseIssuer) processBalanceMessage(message *promise.BalanceMessage) {
	balanceString := fmt.Sprintf("%d%s", message.Balance.Amount, message.Balance.Currency)
	if !message.Accepted {
		seelog.Warn(issuerLogPrefix, fmt.Sprintf("Promise %d is rejected: %s", message.RequestID, balanceString))
	}

	seelog.Info(issuerLogPrefix, fmt.Sprintf("Promise %d balance is %s", message.RequestID, balanceString))
}
