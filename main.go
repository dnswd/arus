package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Statement struct {
	ID     string `json:"id,omitempty"`
	BankID string `json:"bank_id,omitempty"`
	Amount int64  `json:"amount,omitempty"`
}

type Money struct {
	Amount   float64
	Currency string
}

func (m Money) Add(other Money) Money {
	// Add validation for currency consistency if needed
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}
}

func (m Money) Subtract(other Money) Money {
	return Money{Amount: m.Amount - other.Amount, Currency: m.Currency}
}

type AllocationType int

const (
	FixedAmount AllocationType = iota
	Percentage
)

type AllocationRule struct {
	CategoryName string
	Type         AllocationType
	Amount       Money   // Used if Type is FixedAmount
	Percentage   float64 // Used if Type is Percentage (0 to 1)
}

type Category struct {
	Name        string
	Balance     Money
	BankAccount string // Identifier for the bank account
}

func (c *Category) Credit(amount Money) {
	c.Balance = c.Balance.Add(amount)
}

func (c *Category) Debit(amount Money) error {
	if c.Balance.Amount < amount.Amount {
		return fmt.Errorf("insufficient funds in category %s", c.Name)
	}
	c.Balance = c.Balance.Subtract(amount)
	return nil
}

type Expense struct {
	Amount      Money
	Date        time.Time
	Description string
}

type User struct {
	ID              string
	Categories      map[string]*Category
	AllocationRules []AllocationRule
}

func NewUser(id string) *User {
	return &User{
		ID: id,
		Categories: map[string]*Category{
			"Savings":   {Name: "Savings", Balance: Money{Amount: 0, Currency: "USD"}},
			"Emergency": {Name: "Emergency", Balance: Money{Amount: 0, Currency: "USD"}},
			"Expense":   {Name: "Expense", Balance: Money{Amount: 0, Currency: "USD"}},
		},
		AllocationRules: []AllocationRule{},
	}
}

func (u *User) AllocateIncome(income Money) error {
	totalPercentage := 0.0
	totalFixed := Money{Amount: 0, Currency: income.Currency}

	// First pass: calculate total fixed amounts and percentages
	for _, rule := range u.AllocationRules {
		if rule.Type == FixedAmount {
			totalFixed = totalFixed.Add(rule.Amount)
		} else if rule.Type == Percentage {
			totalPercentage += rule.Percentage
		}
	}

	if totalPercentage > 1.0 {
		return errors.New("total allocation percentages exceed 100%")
	}

	if totalFixed.Amount > income.Amount {
		return errors.New("income insufficient for fixed allocations")
	}

	remainingIncome := income.Subtract(totalFixed)

	// Second pass: allocate income to categories
	for _, rule := range u.AllocationRules {
		category, exists := u.Categories[rule.CategoryName]
		if !exists {
			return fmt.Errorf("category %s does not exist", rule.CategoryName)
		}

		var allocation Money
		if rule.Type == FixedAmount {
			allocation = rule.Amount
		} else if rule.Type == Percentage {
			allocationAmount := remainingIncome.Amount * rule.Percentage
			allocation = Money{Amount: allocationAmount, Currency: income.Currency}
		}

		category.Credit(allocation)
	}

	return nil
}

func (u *User) ProcessExpense(expense Expense) error {
	deductionOrder := []string{"Expense", "Emergency", "Savings"}
	amountToDeduct := expense.Amount

	for _, categoryName := range deductionOrder {
		category := u.Categories[categoryName]
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
			deductableAmount := Money{Amount: category.Balance.Amount, Currency: category.Balance.Currency}
			if err := category.Debit(deductableAmount); err != nil {
				return err
			}
			amountToDeduct = amountToDeduct.Subtract(deductableAmount)
		}
	}

	if amountToDeduct.Amount > 0 {
		return errors.New("insufficient funds across all categories")
	}

	return nil
}

type AccountStatement struct {
	BankAccountID string
	Expenses      []Expense
}

func (u *User) ProcessAccountStatement(statement AccountStatement) error {
	// Find the category associated with the bank account
	var category *Category
	for _, c := range u.Categories {
		if c.BankAccount == statement.BankAccountID {
			category = c
			break
		}
	}
	if category == nil {
		return fmt.Errorf("no category associated with bank account %s", statement.BankAccountID)
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
		{CategoryName: "Savings", Type: Percentage, Percentage: 0.2},
		{CategoryName: "Emergency", Type: Percentage, Percentage: 0.3},
		{CategoryName: "Expense", Type: Percentage, Percentage: 0.5},
	}

	income := Money{Amount: 1000, Currency: "USD"}
	err = user.AllocateIncome(income)
	if err != nil {
		fmt.Println("unexpected error: ", err)
	}
	jcart, _ := json.Marshal(user)
	fmt.Println(string(jcart))
}
