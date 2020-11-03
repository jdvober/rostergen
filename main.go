package main

import (
	"encoding/json"
	"fmt"
	"strings"

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
- Remove students from IEP List if they are not in main roster
*/

// SpreadsheetID is the id of the spreadsheet of the Master Roster
const SpreadsheetID string = "1HRfK4yZERLWd-OcDZ8pJRirdzdkHln3SUtIfyGZEjNk"

// Roster is a master list of all students and their relevant information
var Roster = map[string]map[string]string{}
var keyList []string = []string{
	"Last", "First", "SunID", "GoogleID", "CustomID", "Email", "ParentEmail", "GradeLevel", "Course", "IEP", "APEX", "Classroom", "Contacts",
}

func main() {

	/* var wg sync.WaitGroup */
	classroomData := getClassroomData()
	apexData := getAPEXData()
	for _, cd := range classroomData {
		addToRoster(cd)
	}
	for _, ad := range apexData {
		addToRoster(ad)
	}
	// Get data from each paste and Classroom
	// Give each a customID
	// If key of customID is not found in the map for that students, add it, otherwise fill in missing information
	// Erase old data
	// Put new data

	fmt.Println("Number of students on roster: ", len(Roster))
	count := 0
	for _, student := range Roster {
		if count < 50 {
			dataJSON := mssToJSON(student)
			fmt.Println(dataJSON)
		}
		count++
	}

}

func getClassroomData() []map[string]string {
	fmt.Println("Getting student profiles from Google Classroom...")

	data := []map[string]string{}

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
	googleStudentProfiles := make([][]interface{}, len(studentProfiles))

	Counter := 0
	for _, course := range courses {
		students := stu.List(client, course.Id)
		for _, g := range students {
			if course.Name != "Test Class" {
				googleStudentProfiles[Counter] = []interface{}{g.First, g.Last, g.Email, course.Name, g.Id, course.Id}
				cID := makeCustomID(g.Last, g.First)
				profile := map[string]string{
					"Last":      g.Last,
					"First":     g.First,
					"GoogleID":  g.Id,
					"CustomID":  cID,
					"Email":     g.Email,
					"Course":    course.Name,
					"Classroom": "TRUE",
				}
				data = append(data, profile)
				/* fmt.Printf("\nSending %s %s's profile to addToRoster()\n", g.First, g.Last)
				 * addToRoster(profile) */
			}
		}
		Counter++
	}
	return data
}

func getAPEXData() []map[string]string {
	fmt.Println("Getting student profiles from APEX Paste on Google Sheet...")

	data := []map[string]string{}

	// Get Google Client
	client := auth.Authorize()
	// Get the pasted in values from Sunguard
	firstNameRange := "APEX Paste!A2:A"
	lastNameRange := "APEX Paste!B2:B"

	firstNames := sh.Get(client, SpreadsheetID, firstNameRange)
	lastNames := sh.Get(client, SpreadsheetID, lastNameRange)

	// Removing commas and spaces
	for n := range firstNames {
		ln := lastNames[n][0].(string)
		fn := firstNames[n][0].(string)
		cID := makeCustomID(ln, fn)
		profile := map[string]string{
			"Last":     ln,
			"First":    fn,
			"CustomID": cID,
			"APEX":     "TRUE",
		}
		data = append(data, profile)
		/* fmt.Printf("\nSending %s %s's profile to addToRoster()\n", fn, ln)
		 * addToRoster(profile) */
	}
	return data
}

func addToRoster(p map[string]string) {
	tempProfile := map[string]string{
		"Last":        p["Last"],
		"First":       p["First"],
		"SunID":       p["SunID"],
		"GoogleID":    p["GoogleID"],
		"CustomID":    p["CustomID"],
		"Email":       p["Email"],
		"ParentEmail": p["ParentEmail"],
		"GradeLevel":  p["Grade"],
		"Course":      p["Course"],
		"IEP":         p["IEP"],
		"APEX":        p["APEX"],
		"Classroom":   p["Classroom"],
		"Contacts":    p["Contacts"],
	}

	c := tempProfile["CustomID"]
	// Check if student is already in map

	// If student is in map, else
	if profile, ok := Roster[c]; ok {
		// compare roster keys to profile keys.  If not in roster keys, add profile key:value to roster
		fmt.Printf("\n%s %s is already on the roster.  Adding missing information...\n", p["First"], p["Last"])
		for _, key := range keyList {
			// if key from keyList has already been added to the profile
			if val, ok := profile[key]; ok {
				if val == "" {
					if tempProfile[key] != "" {
						fmt.Printf("No value provided yet.  Adding value %q for key %q\n", tempProfile[key], key)
						Roster[c][key] = tempProfile[key]
					}
				} else {
					fmt.Printf("Value %q already stored in key %q\n", val, key)
				}

			} else {
				fmt.Printf("\nNo value in either Roster or provided profile for key %q\n", key)
			}
		}
	} else {
		fmt.Printf("\n%s %s (CustomID = %s) is not on the roster yet.  Adding missing information...\n", tempProfile["First"], tempProfile["Last"], tempProfile["CustomID"])

		Roster[c] = make(map[string]string)
		for _, key := range keyList {
			Roster[c][key] = tempProfile[key]
		}
	}
	// if no, add all info
	// if yes, add info that is missing
}

func makeCustomID(last string, first string) string {
	return strings.Join([]string{strings.ToLower(last), strings.ToUpper(first)}, "")
}

func mssToJSON(m map[string]string) string {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Println(err.Error())
		return "Could Not Convert To JSON"
	}

	jsonStr := string(data)
	return jsonStr
}
