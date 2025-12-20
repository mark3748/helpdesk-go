package main

// ValidTicketStatuses contains all valid ticket status values.
// This is used to validate status changes and maintain consistency across the application.
var ValidTicketStatuses = []string{
	"New",
	"Open",
	"Assigned",
	"Accepted",
	"In Progress",
	"Scheduled",
	"Pending",
	"Pending - Awaiting Info",
	"Pending - Awaiting Callback",
	"Pending - Awaiting Parts",
	"Pending - Awaiting Approval",
	"Resolved",
	"Closed",
}
