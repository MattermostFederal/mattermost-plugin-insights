// Build-time feature flags. These mirror the server-side constants in
// server/api/api.go — flip both together.

// ENABLE_PERSONAL_INSIGHTS mirrors `EnablePersonalInsights` in
// server/api/api.go. Personal ("My") insights are disabled while team
// insights move onto a once-daily server-wide snapshot; the My-scope
// queries run per-request and carry the performance problems catalogued in
// INSIGHTS_REFERENCE.md §3.
//
// With this off the page always renders team scope and the scope selector is
// hidden. Team routes require a Professional+ license, so unlicensed,
// Starter, and non-enterprise builds see nothing. Accepted for now.
export const ENABLE_PERSONAL_INSIGHTS = false;
