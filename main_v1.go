package main

import (
	"fmt"
	"strings"
	"unicode"

	co "github.com/jdvober/goClassroomTools/courses"
	"github.com/jdvober/goClassroomTools/students"
	stu "github.com/jdvober/goClassroomTools/students"
	auth "github.com/jdvober/goGoogleAuth"
	sh "github.com/jdvober/goSheets/values"
)

/*
To Do:
- Check that the sheets actually exist, and if they don't, make them.
*/
type Bio struct {
	First    string
	Last     string
	Email    string
	GoogleID string
	Class    string
}

func main() {

	spreadsheetId := "1HRfK4yZERLWd-OcDZ8pJRirdzdkHln3SUtIfyGZEjNk"
	/* rangeData := "Sheet2!A2" */
	// values := [][]interface{}{{"sample_A1", "sample_B1"}, {"sample_A2", "sample_B2"}, {"sample_A3", "sample_A3"}}
	makeBaseRoster(spreadsheetId)
	checkInSunguard(spreadsheetId)
	fmt.Println("Finished.")
}

func switchGradeLevel(name string) string {

	switch name {
	case "AP Physics", "Physics", "APEX Physics":
		return "12"
	case "APEX Honors Chemistry", "APEX Chemistry":
		return "11"
	case "APEX Physical Science":
		return "9"
	default:
		return ""
	}

}

func checkInSunguard(spreadsheetID string) {
	fmt.Println("Checking if students are in the Sunguard List..")
	// Get Google Client
	client := auth.Authorize()
	// Get the pasted in values from Sunguard
	readRange := "Sunguard Paste!A:A"
	vals := sh.Get(client, spreadsheetID, readRange)
	payload := []string{}
	fmt.Println("Removing Middle Names...")
	for _, name := range vals {
		fullName := name[0].(string)
		// Check to make sure the middle name has not already been removed
		if strings.Count(fullName, " ") > 1 {
			firstCommaLast := strings.TrimRightFunc(fullName, func(run rune) bool {
				return unicode.IsLetter(run)
			})
			firstCommaLast = strings.TrimRightFunc(firstCommaLast, func(run rune) bool {
				return unicode.IsSpace(run)
			})
			// Use type assertion to make interface {} be used as string.  This relies on the data always being a string.
			payload = append(payload, firstCommaLast)
		} else {
			firstCommaLast := fullName
			// Use type assertion to make interface {} be used as string.  This relies on the data always being a string.
			payload = append(payload, firstCommaLast)
		}
	}
	// Convert back to interface
	s := make([]interface{}, len(payload))
	for i, v := range payload {
		s[i] = v
	}

	sh.Update(client, spreadsheetID, readRange, "COLUMNS", s)

}

func makeBaseRoster(spreadsheetId string) {

	fmt.Println("Making Base Roster...")
	client := auth.Authorize()
	courses := co.List(client)
	var studentProfiles []students.Profile

	for _, course := range courses {
		studentList := stu.List(client, course.Id) // CourseId Email Id First Last
		for _, student := range studentList {

			studentProfiles = append(studentProfiles, student)
		}
	}

	// Get all courses
	values := make([][]interface{}, len(studentProfiles))
	counter := 0
	for _, course := range courses {
		students := stu.List(client, course.Id)
		for _, s := range students {
			/* fmt.Println(s.First, s.Last, s.CourseId) */
			gradeLevel := switchGradeLevel(course.Name)
			values[counter] = []interface{}{s.First, s.Last, s.Email, course.Name, gradeLevel, s.Id, course.Id}

			counter++
		}
	}

	sh.BatchUpdate(client, spreadsheetId, "Master!A2:G", "ROWS", values)
}
