package main

func main() {
	// Time series (in single timescale hypertable)
	// 1. period (transaction summary in a month, flows)
	// 2. transaction (list of transaction at the period, statement)

	// User table
	// 1. name
	// 2. username
	// 3. password
	// 4. account-id
	// 5. banks

	// Banks
	// 1. bank-id
	// 2. account-id
	// 3. balance

	// ---

	// Flows
	// 1. income
	// 2. expense
	// 3. emergency fund
	// 4. savings
	// 5. investment
	// And additional information to make this piece of info able to generate
	// sankey diagram.

	// Statement
	// 1. statement-id
	// 2. bank-id
	// 3. amount

	// --- Reconciliation

	// A: Suppose we have bank integration, there should be a way to reconcile
	// account statement difference in bank and system.

	// B: I think we already need reconcile feature at the start, because that's
	// the way we "increase visibility/resolution" in user's financial situation.

	// A: That is true, that means grabbing the account statement from the bank
	// is only a mean to grab user's end balance to reconcile?

	// B: Yes but we also grab bank's account statement too. We want users to match
	// their "percieved" statement and actual bank statement. That way users can
	// compare (ala git diff) their statements to match their bank's statements.

	// A: Good catch, but what if users just don't care to reconcile? The bank's
	// statement is the most viable source of truth after all. So they should be
	// able to just "add the difference" to system's account statement to match
	// bank's account statement.

	// B: Yes, and if said "difference statement" appears in a period, we can nudge
	// users to complete the statements in that period. Or just let them know that
	// there might be inacurrate or missing info.

	// A: Good idea, but we shouldn't make it a warning or something. We want users
	// to optionally use the reconcile feature (opt-in). Gentle notice would suffice.

	// --- Sankey Diagram

	// A: Just checked the requirement for Sankey Diagram, it requires
	// (source, target, value) tuple. So we need a way to classify transaction source
	// of fund (income, emergency, savings, investment).

	// B: This is a difficult problem.

	// C: Hold on a second, didn't we agree that we incrementally use funds in order?
	// For example, if the expense fund still have a balance we will deduct from it.
	// But if it doesn't it will use emergency fund, then saving, then investment.

	// A: That is true, but wait. How do we tell a difference between savings and
	// investment?

	// B: Should we check if user's bank account is a custodian account? If it does
	// then we can deduct from investment?

	// A: That is true, but what if user only have custodian account?

	// C: Maybe we can merge investment and savings?

	// A: Interesting idea, but both of them have different purpose. Savings is for
	// saving money for future use (like purchasing something), and investment is
	// for making money.

	// B: That is true, so the issue is to differentiate between (saving -> expense)
	// and (investment -> expense). In case the investment is used for expense.

	// A: Wait a minute, doesn't investment need to be liquefied before it can be
	// expensed? If so, we can assume all investment flow is (investment -> investment)
	// or (investment -> savings).

	// B: That is correct, and savings can be used for expense.

	// C: So if an investment is liquefied and expensed in the same period, how would
	// the sankey diagram look like?

	// A: Well investment and income will be in the same column. Saving, emergency,
	// and expense in second column. Should be straightforward right?

	// B: Yes, and the income in said period is the income in the last period right?

	// A: I think, but doesn't it makes the users pull the wrong conclusion about
	// period information? That makes period != numerically correct monthly accounting
	// right?

	// C: Should we add timeline info below the sankey diagram? And display the
	// numerically monthly accounting info beside/below sankey diagram?

	// A: That means we should display 2 income flows? One in the income on previous
	// month and one in current month?

	// B: That sounds good, but we need to define a definition of period. Because we
	// record the time series database in 1 row. If we show 2 incomes then how can we
	// put that in a single "period" row?

	// A: Add another column to record previous month's income in current period? Or we can
	// just fetch 2 periods and get previous period income.

	// B: Latter is too confusing, former solution is better. So we need to differentiate
	// between income to be used as current period and income for used in next period.

	// C: This is just too confusing.
}
