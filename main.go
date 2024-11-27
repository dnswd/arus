package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Money struct {
	Amount   decimal.Decimal
	Currency string
}

func (m Money) Add(other Money) Money {
	// Add validation for currency consistency if needed
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}
}

func (m Money) Subtract(other Money) Money {
	return Money{Amount: m.Amount - other.Amount, Currency: m.Currency}
}

type AllocationRule struct {
	CategoryType CategoryType
	Percentage   float64
}

type CategoryType int

const (
	Expense CategoryType = iota
	Emergency
	Savings
)

func (c CategoryType) String() string {
	return [...]string{"Expense", "Emergency", "Savings"}[c]
}

type BankAccount struct {
	AccountNumber string
	BankName      string
}

type Category struct {
	Type        CategoryType
	Balance     Money
	BankAccount BankAccount
}

func (c *Category) Credit(amount Money) {
	c.Balance = c.Balance.Add(amount)
}

func (c *Category) Debit(amount Money) error {
	if c.Balance.Amount < amount.Amount {
		return fmt.Errorf("insufficient funds in category %s", c.Type.String())
	}
	c.Balance = c.Balance.Subtract(amount)
	return nil
}

type Transaction struct {
	Amount      Money
	Date        time.Time
	Description string
}

type User struct {
	ID              string
	Categories      map[CategoryType]*Category
	AllocationRules []AllocationRule
}

func NewUser(id string) *User {
	return &User{
		ID: id,
		Categories: map[CategoryType]*Category{
			Expense: {
				Type:    Expense,
				Balance: Money{Amount: 0, Currency: "USD"},
				BankAccount: BankAccount{
					AccountNumber: "EXP123",
					BankName:      "Expense Bank",
				},
			},
			Emergency: {
				Type:    Emergency,
				Balance: Money{Amount: 0, Currency: "USD"},
				BankAccount: BankAccount{
					AccountNumber: "EMG123",
					BankName:      "Emergency Bank",
				},
			},
			Savings: {
				Type:    Savings,
				Balance: Money{Amount: 0, Currency: "USD"},
				BankAccount: BankAccount{
					AccountNumber: "SAV123",
					BankName:      "Savings Bank",
				},
			},
		},
		AllocationRules: []AllocationRule{},
	}
}

func (u *User) AllocateIncome(income Money) error {
	totalPercentage := 0.0

	if len(u.AllocationRules) < 1 {
		return errors.New("user does not have allocation planned")
	}

	// Calculate total percentages
	for _, rule := range u.AllocationRules {
		totalPercentage += rule.Percentage
	}

	if totalPercentage > 1.0 {
		return errors.New("total allocation percentages exceed 100%")
	}

	// Allocate income to categories
	for _, rule := range u.AllocationRules {
		category, exists := u.Categories[rule.CategoryType]
		if !exists {
			return fmt.Errorf("category %s does not exist", rule.CategoryType.String())
		}

		allocationAmount := income.Amount * rule.Percentage
		allocation := Money{Amount: allocationAmount, Currency: income.Currency}
		category.Credit(allocation)
	}
	return nil
}

func (u *User) ProcessExpense(expense Transaction) error {
	deductionOrder := []CategoryType{Expense, Emergency, Savings}
	amountToDeduct := expense.Amount

	for _, categoryType := range deductionOrder {
		category := u.Categories[categoryType]
		if category == nil {
			continue
		}

		if category.Balance.Amount >= amountToDeduct.Amount {
			if err := category.Debit(amountToDeduct); err != nil {
				return err
			}
			amountToDeduct = Money{Amount: 0, Currency: amountToDeduct.Currency}
			break
		} else {
			deductibleAmount := Money{Amount: category.Balance.Amount, Currency: category.Balance.Currency}
			if err := category.Debit(deductibleAmount); err != nil {
				return err
			}
			amountToDeduct = amountToDeduct.Subtract(deductibleAmount)
		}
	}

	if amountToDeduct.Amount > 0 {
		return errors.New("insufficient funds across all categories")
	}

	return nil
}

type AccountStatement struct {
	BankAccount BankAccount
	Expenses    []Transaction
}

func (u *User) ProcessAccountStatement(statement AccountStatement) error {
	// Find the category associated with the bank account
	var category *Category
	for _, c := range u.Categories {
		if c.BankAccount.AccountNumber == statement.BankAccount.AccountNumber &&
			c.BankAccount.BankName == statement.BankAccount.BankName {
			category = c
			break
		}
	}
	if category == nil {
		return fmt.Errorf("no category associated with bank account %s at %s",
			statement.BankAccount.AccountNumber, statement.BankAccount.BankName)
	}

	// Process each expense
	for _, expense := range statement.Expenses {
		if err := u.ProcessExpense(expense); err != nil {
			return err
		}
	}
	return nil
}

type UserRepository interface {
	GetByID(id string) (*User, error)
	Save(user *User) error
}

type InMemoryUserRepository struct {
	data map[string]*User
	mu   sync.RWMutex
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		data: make(map[string]*User),
	}
}

func (r *InMemoryUserRepository) GetByID(id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.data[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (r *InMemoryUserRepository) Save(user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[user.ID] = user
	return nil
}

type FinanceService struct {
	UserRepo UserRepository
}

func (s *FinanceService) AllocateIncome(ctx context.Context, userID string, income Money) error {
	user, err := s.UserRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if err := user.AllocateIncome(income); err != nil {
		return err
	}

	return s.UserRepo.Save(user)
}

func (s *FinanceService) ProcessAccountStatement(ctx context.Context, userID string, statement AccountStatement) error {
	user, err := s.UserRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if err := user.ProcessAccountStatement(statement); err != nil {
		return err
	}

	return s.UserRepo.Save(user)
}

func main() {
	repo := NewInMemoryUserRepository()

	// Create a new user
	user := NewUser("user123")
	fmt.Println("Creating user:", user.ID)
	if err := repo.Save(user); err != nil {
		fmt.Println("Error saving user:", err)
		return
	}

	// Retrieve the user
	retrievedUser, err := repo.GetByID("user123")
	if err != nil {
		fmt.Println("Error retrieving user:", err)
		return
	}
	fmt.Println("Retrieved user ID:", retrievedUser.ID)

	user.AllocationRules = []AllocationRule{
		{CategoryType: Expense, Percentage: 0.5},
		{CategoryType: Emergency, Percentage: 0.3},
		{CategoryType: Savings, Percentage: 0.2},
	}

	income := Money{Amount: 1000, Currency: "USD"}
	err = user.AllocateIncome(income)
	if err != nil {
		fmt.Println("unexpected error: ", err)
	}

	jcart, _ := json.Marshal(user)
	fmt.Println(string(jcart))

	expense := Transaction{
		Amount:      Money{Amount: 900, Currency: "USD"},
		Description: "Unexpected Expense",
	}

	err = user.ProcessExpense(expense)
	if err != nil {
		fmt.Printf("unexpected error: %v", err)
	}

	jcart, _ = json.Marshal(user)
	fmt.Println(string(jcart))
}
