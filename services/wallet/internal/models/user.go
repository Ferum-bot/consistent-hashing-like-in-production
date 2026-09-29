package models

import "slices"

// User — пользователь и все его кошельки.
// Шардируем по пользователю: все его кошельки всегда лежат на одной ноде.
type User struct {
	ID      int64
	Wallets []Wallet
}

// Wallet — кошелёк в одной валюте. Баланс в минимальных единицах валюты: центах, копейках, иенах.
type Wallet struct {
	Currency string
	Balance  int64
}

// Clone возвращает копию, у которой свой слайс кошельков.
func (u User) Clone() User {
	u.Wallets = slices.Clone(u.Wallets)

	return u
}

// FindWallet находит кошелёк в нужной валюте. Если такого нет — nil.
func (u *User) FindWallet(currency string) *Wallet {
	for i := range u.Wallets {
		if u.Wallets[i].Currency == currency {
			return &u.Wallets[i]
		}
	}

	return nil
}

// OpenWallet находит кошелёк в нужной валюте, а если его нет — заводит пустой.
func (u *User) OpenWallet(currency string) *Wallet {
	if wallet := u.FindWallet(currency); wallet != nil {
		return wallet
	}

	u.Wallets = append(u.Wallets, Wallet{Currency: currency})

	return &u.Wallets[len(u.Wallets)-1]
}

// Stats — сколько данных лежит на ноде. Отсюда видно перекос между нодами.
type Stats struct {
	Users   int
	Wallets int
}
