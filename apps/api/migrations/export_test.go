package migrations

// UpFrom is Up with other migrations than the embedded ones, for the tests in
// package migrations_test (which can't be in this one: pgtest imports it).
var UpFrom = up
