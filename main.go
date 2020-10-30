package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	co "github.com/jdvober/goClassroomTools/courses"
	"github.com/jdvober/goClassroomTools/students"
	stu "github.com/jdvober/goClassroomTools/students"
	auth "github.com/jdvober/goGoogleAuth"
	sh "github.com/jdvober/goSheets/values"
)

/*
TODO:
- Check that the sheets actually exist, and if they don't, make them.
- Add students from APEX and Sunguard that are not in the Google Classroom Roster
- Make a "TO DO" list for adding students to classroom, contacts etc.
- Need to add error checking
- Google Classroom Gradebook Integration
- Add Convert Back To Interface function to goSheets
- Add Clear Cells function to goSheets
- Add Parent Emails
- Add Date Added column
*/

const SpreadsheetID string = "1HRfK4yZERLWd-OcDZ8pJRirdzdkHln3SUtIfyGZEjNk"

func main() {

	var wg sync.WaitGroup

	// Make the base roster from Google Classroom Data and wait for it to finish, since further functions depend on this data.
	wg.Add(1)
	go makeBaseRoster(&wg)
	wg.Wait()

	// Make a list of formatted names and post
	wg.Add(1)
	go makeFormattedNames(&wg)
	wg.Wait()

	// Run concurrent tasks
	wg.Add(1)
	go checkInSunguard(&wg)
	wg.Add(1)
	go checkInAPEX(&wg)
	wg.Add(1)
	go checkInIEP(&wg)
	wg.Wait()

	fmt.Println("Finished.")
}

func makeBaseRoster(wg *sync.WaitGroup) {
	defer wg.Done()
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
	valuesBlank := make([][]interface{}, 300)

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
	// Make a blank set of data to use to overwrite the old data (This could probably be done a lot cleaner)
	for count := 0; count < 300; count++ {
		valuesBlank[count] = []interface{}{"", "", "", "", "", "", ""}
	}

	// Clearning the cells
	sh.BatchUpdate(client, SpreadsheetID, "Master!A2:G", "ROWS", valuesBlank)
	// Write new roster
	sh.BatchUpdate(client, SpreadsheetID, "Master!A2:G", "ROWS", values)

	inClassroom := make([]string, len(values), len(values))
	/*     inClassroom := []string{}
	 *
	 *     for c, _ := range values {
	 *         if c == 0 {
	 *             fmt.Println("\n")
	 *         }
	 *         inClassroom = append(inClassroom, "")
	 *     } */

	x := make([]interface{}, len(inClassroom))
	for i, v := range inClassroom {
		x[i] = v
	}

	// Clear columns
	clearColumn("Master!M2:M")

	// Write FALSE to cells corresponding with students who are NOT on the Google Classroom roster
	sh.Update(client, SpreadsheetID, "Master!M2:M", "COLUMNS", x)

	fmt.Println("Done making Base Roster.")
}

func makeFormattedNames(wg *sync.WaitGroup) {
	defer wg.Done()
	// TODO getColumn()

	// Get Google Client
	client := auth.Authorize()
	// Get the names from the master roster's first two columns
	readRange := "Master!A2:B"
	vals := sh.Get(client, SpreadsheetID, readRange)

	payload1 := []string{}
	payload2 := []string{}
	payload3 := []string{}

	for _, name := range vals {
		firstName := name[0].(string)
		lastName := name[1].(string)

		lastCommaFirst := lastName + ", " + firstName
		firstLast := firstName + "" + lastName
		myIDFormat := strings.ToLower(lastName) + firstName

		payload1 = append(payload1, lastCommaFirst)
		payload2 = append(payload2, firstLast)
		payload3 = append(payload3, myIDFormat)
	}
	// TODO update names in correct format

	// Convert back to interface
	/* s := make([]interface{}, len(payload1))
	 * for i, v := range payload1 {
	 *     s[i] = v
	 * } */
	/* t := make([]interface{}, len(payload2))
	 * for i, v := range payload2 {
	 *     t[i] = v
	 * }
	 * u := make([]interface{}, len(payload3))
	 * for i, v := range payload3 {
	 *     u[i] = v
	 * } */

	// Convert them back to interfaces
	s := toInterface(&payload1)
	t := toInterface(&payload2)
	u := toInterface(&payload3)

	// Clear columns
	clearColumn("Master!O2:O")
	clearColumn("Master!P2:P")
	clearColumn("Master!Q2:Q")

	// Write Last, First
	sh.Update(client, SpreadsheetID, "Master!O2", "COLUMNS", s)
	// Write First Last
	sh.Update(client, SpreadsheetID, "Master!P2", "COLUMNS", t)
	// Write masterMyCustomIDs
	sh.Update(client, SpreadsheetID, "Master!Q2", "COLUMNS", u)
	fmt.Println("Done writing Formatted Names to sheet.")
}

func checkInSunguard(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Checking if students are in the Sunguard List..")
	// Get Google Client
	client := auth.Authorize()
	// Get the pasted in values from Sunguard
	readRange := "Sunguard Paste!A2:A"
	vals := sh.Get(client, SpreadsheetID, readRange)
	payload := []string{}

	for _, name := range vals {
		fullName := name[0].(string)
		// Check to make sure the middle name has not already been removed
		if strings.Count(fullName, "") > 1 {
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

	// Clear columns
	clearColumn(readRange)

	// Posting cleaned names to sheet
	sh.Update(client, SpreadsheetID, readRange, "COLUMNS", s)

	payload2 := []string{}
	// Removing commas and spaces
	for _, name := range payload {
		nameSplit := strings.SplitAfter(name, ",")
		nameSplit[0] = strings.TrimRight(nameSplit[0], ",")
		nameSplit[0] = strings.TrimSpace(nameSplit[0])
		nameSplit[0] = strings.ToLower(nameSplit[0])
		nameSplit[1] = strings.TrimSpace(nameSplit[1])

		payload2 = append(payload2, nameSplit[0]+nameSplit[1])
	}
	// Convert back to interface
	t := make([]interface{}, len(payload2))
	for i, v := range payload2 {
		t[i] = v
	}

	// Clear columns
	clearColumn("Sunguard Paste!E2:E")

	// Adding myIDFormat values to "SunguardPaste!E:E"
	sh.Update(client, SpreadsheetID, "Sunguard Paste!E2:E", "COLUMNS", t)

	// Post TRUE to correct cell if myIDFormats match
	masterMyCustomIDs := sh.Get(client, SpreadsheetID, "Master!Q2:Q")
	cusIDAndSID := sh.Get(client, SpreadsheetID, "Sunguard Paste!B2:E")

	payload3 := []string{}
	payload4 := []string{}

	for _, mID := range masterMyCustomIDs {
		foundMatch := false
		foundMatchIndex := 0

		for sID, scID := range payload2 {
			if mID[0].(string) == scID {
				foundMatch = true
				foundMatchIndex = sID
				break
			}
		}

		if foundMatch == true {
			payload3 = append(payload3, "")
			payload4 = append(payload4, cusIDAndSID[foundMatchIndex][0].(string))
		} else {
			payload3 = append(payload3, "FALSE")
			payload4 = append(payload4, "")
		}
	}

	u := make([]interface{}, len(payload3))
	for i, v := range payload3 {
		u[i] = v
	}

	a := make([]interface{}, len(payload4))
	for i, v := range payload4 {
		a[i] = v
	}

	// Clear columns
	clearColumn("Master!K2:K")
	clearColumn("Master!H2:H")

	// Post FALSE if a student is not found on the Sunguard list
	sh.Update(client, SpreadsheetID, "Master!K2:K", "COLUMNS", u)
	// Post Sunguard IDs
	sh.Update(client, SpreadsheetID, "Master!H2:H", "COLUMNS", a)

	fmt.Println("Done checking if students are in the Sunguard List.")
}

func checkInAPEX(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Checking if students are in the APEX List..")
	// Get Google Client
	client := auth.Authorize()
	// Get the pasted in values from Sunguard
	/* readRange := "APEX Paste!A2:B" */
	firstNameRange := "APEX Paste!A2:A"
	lastNameRange := "APEX Paste!B2:B"

	/* vals := sh.Get(client, SpreadsheetID, readRange) */
	firstNames := sh.Get(client, SpreadsheetID, firstNameRange)
	lastNames := sh.Get(client, SpreadsheetID, lastNameRange)

	payload := []string{}

	// Removing commas and spaces
	for n, name := range firstNames {
		ln := strings.ToLower(lastNames[n][0].(string))

		payload = append(payload, ln+name[0].(string))
	}
	// Convert back to interface
	t := make([]interface{}, len(payload))
	for i, v := range payload {
		t[i] = v
	}
	// Clear columns
	clearColumn("APEX Paste!E2:E")
	// Adding myIDFormat values to "APEX Paste!E:E"
	sh.Update(client, SpreadsheetID, "APEX Paste!E2:E", "COLUMNS", t)

	// Post TRUE to correct cell if myIDFormats match
	masterMyCustomIDs := sh.Get(client, SpreadsheetID, "Master!Q2:Q")

	payload2 := []string{}

	for _, mID := range masterMyCustomIDs {
		foundMatch := false
		for _, aID := range payload {
			if mID[0].(string) == aID {
				foundMatch = true
				break
			}
		}

		if foundMatch == true {
			payload2 = append(payload2, "")
		} else {
			payload2 = append(payload2, "FALSE")
		}
	}

	u := make([]interface{}, len(payload2))
	for i, v := range payload2 {
		u[i] = v
	}

	// Clear columns
	clearColumn("Master!L2:L")

	// Post False if the student is not found in the APEX list
	sh.Update(client, SpreadsheetID, "Master!L2:L", "COLUMNS", u)

	fmt.Println("Done checking if students are in the APEX List.")
}

func checkInIEP(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Checking if students have an IEP or 504...")

	// Get Google Client
	client := auth.Authorize()

	readRange := "IEP Paste!B10:B"
	vals := sh.Get(client, SpreadsheetID, readRange)

	payload := []string{}

	// Removing commas and spaces
	for _, name := range vals {
		/* nameSplit := strings.Split(name[0].(string), ",") */

		ln := strings.TrimSpace(strings.Split(name[0].(string), ",")[0])
		fn := strings.TrimSpace(strings.Split(name[0].(string), ",")[1])
		/* ln := strings.TrimSpace(nameSplit[0]) */
		/* fn := strings.TrimSpace(nameSplit[1]) */
		myIDFormat := strings.ToLower(ln) + fn

		payload = append(payload, myIDFormat)
	}
	// Convert back to interface
	s := make([]interface{}, len(payload))
	for i, v := range payload {
		s[i] = v
	}

	// Clear columns
	clearColumn("IEP Paste!S10:S")

	// Adding myIDFormat values to range
	sh.Update(client, SpreadsheetID, "IEP Paste!S10:S", "COLUMNS", s)

	masterMyCustomIDs := sh.Get(client, SpreadsheetID, "Master!Q2:Q")

	payload2 := []string{}

	for _, mID := range masterMyCustomIDs {
		foundMatch := false
		for _, IEPID := range payload {
			if mID[0].(string) == IEPID {
				foundMatch = true
				break
			}
		}

		if foundMatch == true {
			payload2 = append(payload2, "TRUE")
		} else {
			payload2 = append(payload2, "")
		}
	}

	t := make([]interface{}, len(payload2))
	for i, v := range payload2 {
		t[i] = v
	}

	// Clear columns
	clearColumn("Master!J2:J")

	// Post TRUE to cells corresponging with students who have an IEP
	sh.Update(client, SpreadsheetID, "Master!J2:J", "COLUMNS", t)

	fmt.Println("Done checking if students have an IEP.")
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

func clearColumn(writeRange string) {
	// Get Google Client
	client := auth.Authorize()
	vals := sh.Get(client, SpreadsheetID, writeRange)
	/* payload := []string{} */

	payload := make([]string, len(vals), len(vals))
	/* // Create a blank set of cells
	 * for _, row := range vals {
	 *     payload = append(payload, "")
	 * } */

	// Convert back to interface
	t := make([]interface{}, len(payload))
	for i, v := range payload {
		t[i] = v
	}

	// Clearning the cells
	sh.Update(client, SpreadsheetID, writeRange, "COLUMNS", t)
}

func toInterface(payload *[]string) []interface{} {
	s := make([]interface{}, len(*payload))
	for i, v := range *payload {
		s[i] = v
	}
	return s
}
