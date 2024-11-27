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

// Money
type Money struct {
	Amount   decimal.Decimal
	Currency string
}

func NewMoney(amount decimal.Decimal, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: currency,
	}
}

func NewMoneyZero(currency string) Money {
	return Money{
		Amount:   decimal.Zero,
		Currency: currency,
	}
}

func (m Money) Add(other Money) Money {
	// Add validation for currency consistency if needed
	return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}
}

func (m Money) Subtract(other Money) Money {
	if other.IsNegative() {
		return Money{Amount: m.Amount.Sub(other.Amount.Abs()), Currency: m.Currency}
	}
	return Money{Amount: m.Amount.Sub(other.Amount), Currency: m.Currency}
}

func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

func (m Money) IsNegative() bool {
	return m.Amount.IsNegative()
}

// Category type
type CategoryType int

const (
	Expense CategoryType = iota
	Emergency
	Savings
)

func (c CategoryType) String() string {
	return [...]string{"Expense", "Emergency", "Savings"}[c]
}

// Allocation Rule
type AllocationRule struct {
	CategoryType CategoryType
	Percentage   decimal.Decimal
}

// Bank
type BankAccount struct {
	AccountNumber string
	BankName      string
}

// User's Category
type Category struct {
	Type        CategoryType
	Balance     Money
	BankAccount BankAccount
}

func (c *Category) Credit(amount Money) {
	c.Balance = c.Balance.Add(amount)
}

func (c *Category) Debit(amount Money) error {
	if c.Balance.Amount.LessThan(amount.Amount) {
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

func NewTransaction(amount Money, date time.Time, description string) Transaction {
	return Transaction{
		Amount:      amount,
		Date:        date,
		Description: description,
	}
}

func NewIncome(amount Money, date time.Time, description string) Transaction {
	return Transaction{
		Amount:      amount,
		Date:        date,
		Description: description,
	}
}

func NewExpense(amount Money, date time.Time, description string) Transaction {
	return Transaction{
		Amount:      Money{Amount: amount.Amount.Neg(), Currency: amount.Currency},
		Date:        date,
		Description: description,
	}
}

type User struct {
	ID              string
	Categories      map[CategoryType]*Category
	AllocationRules []AllocationRule
	Incomes         []Transaction
	Expenses        []Transaction
}

func NewUser(id string) *User {
	return &User{
		ID: id,
		Categories: map[CategoryType]*Category{
			Expense: {
				Type:    Expense,
				Balance: NewMoneyZero("USD"),
				BankAccount: BankAccount{
					AccountNumber: "EXP123",
					BankName:      "Expense Bank",
				},
			},
			Emergency: {
				Type:    Emergency,
				Balance: NewMoneyZero("USD"),
				BankAccount: BankAccount{
					AccountNumber: "EMG123",
					BankName:      "Emergency Bank",
				},
			},
			Savings: {
				Type:    Savings,
				Balance: NewMoneyZero("USD"),
				BankAccount: BankAccount{
					AccountNumber: "SAV123",
					BankName:      "Savings Bank",
				},
			},
		},
		AllocationRules: []AllocationRule{},
		Incomes:         []Transaction{},
		Expenses:        []Transaction{},
	}
}

func (u *User) AllocateIncome(income Money, date time.Time, description string) error {
	totalPercentage := decimal.Zero

	if len(u.AllocationRules) < 1 {
		return errors.New("user does not have allocation planned")
	}

	// Calculate total percentages
	for _, rule := range u.AllocationRules {
		totalPercentage = totalPercentage.Add(rule.Percentage)
	}

	if totalPercentage.GreaterThan(decimal.NewFromInt(1)) {
		return errors.New("total allocation percentages exceed 100%")
	}

	// Allocate income to categories
	for _, rule := range u.AllocationRules {
		category, exists := u.Categories[rule.CategoryType]
		if !exists {
			return fmt.Errorf("category %s does not exist", rule.CategoryType.String())
		}

		allocationAmount := income.Amount.Mul(rule.Percentage)
		allocation := Money{Amount: allocationAmount, Currency: income.Currency}
		category.Credit(allocation)
	}

	// Record the income
	newIncome := NewTransaction(income, date, description)
	u.Incomes = append(u.Incomes, newIncome)

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

		if category.Balance.Amount.GreaterThanOrEqual(amountToDeduct.Amount) {
			if err := category.Debit(amountToDeduct); err != nil {
				return err
			}
			amountToDeduct = Money{Amount: decimal.Zero, Currency: amountToDeduct.Currency}
			break
		} else {
			deductibleAmount := Money{Amount: category.Balance.Amount, Currency: category.Balance.Currency}
			if err := category.Debit(deductibleAmount); err != nil {
				return err
			}
			amountToDeduct = amountToDeduct.Subtract(deductibleAmount)
		}
	}

	if amountToDeduct.Amount.GreaterThan(decimal.Zero) {
		return errors.New("insufficient funds across all categories")
	}

	u.Expenses = append(u.Expenses, expense)

	return nil
}

func (u *User) GetPeriodSummary(period Period) (Money, []Transaction, Money, []Transaction) {
	totalExpense := NewMoneyZero("USD")
	var expensesInPeriod []Transaction

	for _, expense := range u.Expenses {
		if period.Contains(expense.Date) {
			totalExpense = totalExpense.Add(expense.Amount)
			expensesInPeriod = append(expensesInPeriod, expense)
		}
	}

	totalIncome := NewMoneyZero("USD")
	var incomesInPeriod []Transaction

	for _, income := range u.Incomes {
		if period.Contains(income.Date) {
			totalIncome = totalIncome.Add(income.Amount)
			incomesInPeriod = append(incomesInPeriod, income)
		}
	}

	return totalExpense, expensesInPeriod, totalIncome, incomesInPeriod
}

func (u *User) CheckIncomeStatus(period Period) (string, error) {
	totalExpense, _, totalIncome, _ := u.GetPeriodSummary(period)

	// Check if Emergency or Savings funds were used
	emergencyUsed := decimal.Zero.Sub(u.Categories[Emergency].Balance.Amount).GreaterThan(decimal.Zero)
	savingsUsed := decimal.Zero.Sub(u.Categories[Savings].Balance.Amount).GreaterThan(decimal.Zero)

	if emergencyUsed || savingsUsed {
		warning := "Warning: You have used "
		if emergencyUsed {
			warning += "Emergency funds "
		}
		if savingsUsed {
			if emergencyUsed {
				warning += "and "
			}
			warning += "Savings funds "
		}
		warning += "to cover your expenses. Consider adjusting your lifestyle or increasing your income."
		return warning, nil
	}

	if totalIncome.Amount.GreaterThanOrEqual(totalExpense.Amount) {
		return "Your income covers your expenses.", nil
	} else {
		return "Your expenses exceed your income.", nil
	}
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

type Period struct {
	StartDate time.Time
	EndDate   time.Time
}

func (p Period) Contains(date time.Time) bool {
	return !date.Before(p.StartDate) && !date.After(p.EndDate)
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

	if err := user.AllocateIncome(income, time.Now(), ""); err != nil {
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

func CreateMonthlyPeriod(year int, month time.Month) Period {
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)
	return Period{
		StartDate: startDate,
		EndDate:   endDate,
	}
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
		{CategoryType: Expense, Percentage: decimal.NewFromFloat(0.5)},
		{CategoryType: Emergency, Percentage: decimal.NewFromFloat(0.3)},
		{CategoryType: Savings, Percentage: decimal.NewFromFloat(0.2)},
	}

	period := CreateMonthlyPeriod(2023, time.September)

	income := Money{Amount: decimal.NewFromInt(1000), Currency: "USD"}
	err = user.AllocateIncome(income, time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC), "September Salary")
	if err != nil {
		fmt.Println("unexpected error: ", err)
	}

	jcart, _ := json.Marshal(user)
	fmt.Println(string(jcart))

	expenseAmount := Money{Amount: decimal.NewFromInt(900), Currency: "USD"}
	expense := NewExpense(expenseAmount, time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC), "Car Repair")
	user.ProcessExpense(expense)

	err = user.ProcessExpense(expense)
	if err != nil {
		fmt.Printf("unexpected error: %v", err)
	}

	jcart, _ = json.Marshal(user)
	fmt.Println(string(jcart))

	// Get expense summary
	totalExpense, expenses, totalIncome, incomes := user.GetPeriodSummary(period)
	fmt.Printf("Total Expenses: %s\n", totalExpense.Amount.StringFixed(2))
	for _, e := range expenses {
		fmt.Printf(" - %s: %s on %s\n", e.Description, e.Amount.Amount.StringFixed(2), e.Date.Format("2006-01-02"))
	}

	// Get income summary
	fmt.Printf("Total Income: %s\n", totalIncome.Amount.StringFixed(2))
	for _, i := range incomes {
		fmt.Printf(" - %s: %s on %s\n", i.Description, i.Amount.Amount.StringFixed(2), i.Date.Format("2006-01-02"))
	}

	// TODO: Income status masih ga bener, need to check parity control

	// Check income status
	status, err := user.CheckIncomeStatus(period)
	if err != nil {
		fmt.Println("Error checking income status:", err)
	} else {
		fmt.Println("Income Status:", status)
	}
}
