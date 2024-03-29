package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/jdvober/gauth"
	"github.com/jdvober/gclass"
	"github.com/jdvober/gsheets"
)

/*
TODO:
- Check that the sheets actually exist, and if they don't, make them.
- Make a "TO DO" list for adding students to classroom, contacts etc.
- Need to add error checking
- Google Classroom Gradebook Integration
- Add Convert Back To Interface function to goSheets
- Add Clear Cells function to goSheets
- Add Date Added column
- Remove students from IEP List if they are not in main roster
*/

// SpreadsheetID is the id of the spreadsheet of the Master Roster
const SpreadsheetID string = "1HRfK4yZERLWd-OcDZ8pJRirdzdkHln3SUtIfyGZEjNk"

// Roster is a master list of all students and their relevant information
var Roster = map[string]map[string]string{}
var keyList []string = []string{
	"Last", "First", "SunID", "GoogleID", "GoogleCourseID", "CustomID", "Email", "GradeLevel", "Mod", "Course", "IEP", "Classroom", "Sunguard",
}

func main() {
	// Create log file
	// If the file doesn't exist, create it or append to the file
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	// Get data sources
	classroomData := getClassroomData()
	sunguardData := getSunguardData()
	iepData := getIEPData()

	for _, cd := range classroomData {
		addToRoster(cd)
	}
	for _, sd := range sunguardData {
		addToRoster(sd)
	}
	for _, iepd := range iepData {
		addToRoster(iepd)
	}

	fmt.Printf("\nPosting to sheet...")
	PostToSheet(Roster)
	fmt.Printf(" Done")
}

func getClassroomData() []map[string]string {
	fmt.Printf("\nGetting student profiles from Google Classroom...")

	data := []map[string]string{}

	client := gauth.Authorize()
	courses := gclass.ListCourses(client)
	var spreadsheetProfiles []gclass.Profile

	for _, course := range courses {
		studentList := gclass.ListStudents(client, course.Id) // CourseId Email Id First Last
		for _, student := range studentList {
			spreadsheetProfiles = append(spreadsheetProfiles, student)
		}
	}

	// Get all courses
	googleStudentProfiles := make([][]interface{}, len(spreadsheetProfiles))

	Counter := 0
	for _, course := range courses {
		students := gclass.ListStudents(client, course.Id)
		for _, g := range students {
			if course.Name != "Test Class" {
				googleStudentProfiles[Counter] = []interface{}{g.First, g.Last, g.Email, course.Name, g.Id, course.Id}
				cID := makeCustomID(g.Last, g.First)
				profile := map[string]string{
					"Last":           g.Last,
					"First":          g.First,
					"GoogleID":       g.Id,
					"GoogleCourseID": course.Id,
					"CustomID":       cID,
					"Email":          g.Email,
					"Course":         course.Name,
					"Classroom":      "TRUE",
				}
				data = append(data, profile)
				/* log.Printf("\nSending %s %s's profile to addToRoster()\n", g.First, g.Last)
				 * addToRoster(profile) */
			}
		}
		Counter++
	}
	fmt.Printf(" Done")
	return data
}

func getSunguardData() []map[string]string {
	fmt.Printf("\nGetting student profiles from Sunguard Paste on Google Sheet...")

	data := []map[string]string{}

	// Get Google Client
	client := gauth.Authorize()
	// Get the pasted in values from Sunguard
	readRange := "Sunguard Paste!A2:D"
	vals := gsheets.GetValues(client, SpreadsheetID, readRange)

	fullNames := []string{}
	sunguardIDs := []string{}
	/* sunguardCourseIDs := []string{} */
	courses := []string{}
	mods := []string{}
	gLevels := []string{}
	for _, name := range vals {
		fullName := name[0].(string)
		sunguardID := name[1].(string)
		sunguardCourseID := name[2].(string)
		gLevel := name[3].(string)

		// Check to make sure the middle name has not already been removed
		if strings.Count(fullName, "") > 1 {
			firstCommaLast := strings.TrimRightFunc(fullName, func(run rune) bool {
				return unicode.IsLetter(run)
			})
			firstCommaLast = strings.TrimRightFunc(firstCommaLast, func(run rune) bool {
				return unicode.IsSpace(run)
			})
			// Use type assertion to make interface {} be used as string.  This relies on the data always being a string.
			fullNames = append(fullNames, firstCommaLast)
		} else {
			firstCommaLast := fullName
			// Use type assertion to make interface {} be used as string.  This relies on the data always being a string.
			fullNames = append(fullNames, firstCommaLast)
		}

		// Add Sunguard IDs to slice
		sunguardIDs = append(sunguardIDs, sunguardID)

		// Convert Sunguard Course IDs to Course Name and Mod
		switch sunguardCourseID {
		case "2020-23":
			courses = append(courses, "Academic Assist")
			mods = append(mods, "4")
		case "0230-1":
			courses = append(courses, "Physics")
			mods = append(mods, "2")
		case "0232-1":
			courses = append(courses, "AP Physics")
			mods = append(mods, "3")
		case "0209-6":
			courses = append(courses, "Physical Science")
			mods = append(mods, "5")
		case "0209-7":
			courses = append(courses, "Physical Science")
			mods = append(mods, "7")
		case "0209-8":
			courses = append(courses, "Physical Science")
			mods = append(mods, "8")
		case "0209-9":
			courses = append(courses, "Physical Science")
			mods = append(mods, "9")
		default:
			courses = append(courses, "")
			mods = append(mods, "")
		}

		// Add GradeLevel to slice
		gLevels = append(gLevels, gLevel)

	}

	// Removing commas and spaces
	for n, name := range fullNames {
		nameSplit := strings.SplitAfter(name, ",")
		nameSplit[0] = strings.TrimRight(nameSplit[0], ",")
		nameSplit[0] = strings.TrimSpace(nameSplit[0])
		nameSplit[1] = strings.TrimSpace(nameSplit[1])
		nameSplit[1] = strings.Split(nameSplit[1], " ")[0]

		fullNames[n] = nameSplit[0] + nameSplit[1]
		cID := makeCustomID(nameSplit[0], nameSplit[1])
		profile := map[string]string{
			"Last":       nameSplit[0],
			"First":      nameSplit[1],
			"CustomID":   cID,
			"Sunguard":   "TRUE",
			"Course":     courses[n],
			"Mod":        mods[n],
			"GradeLevel": gLevels[n],
			"SunID":      sunguardIDs[n],
		}
		data = append(data, profile)
	}
	/* log.Printf("\nSending %s %s's profile to addToRoster()\n", fn, ln)
	 * addToRoster(profile) */
	fmt.Printf(" Done")
	return data
}

func getIEPData() []map[string]string {
	fmt.Printf("\nChecking if students have an IEP or 504...")

	// Get Google Client
	client := gauth.Authorize()

	readRange := "IEP List!B10:B"
	vals := gsheets.GetValues(client, SpreadsheetID, readRange)

	data := []map[string]string{}

	// Removing commas and spaces
	for _, name := range vals {

		ln := strings.TrimSpace(strings.Split(name[0].(string), ",")[0])
		fn := strings.TrimSpace(strings.Split(name[0].(string), ",")[1])
		cID := makeCustomID(ln, fn)

		profile := map[string]string{
			"Last":     ln,
			"First":    fn,
			"CustomID": cID,
			"IEP":      "TRUE",
		}
		data = append(data, profile)
	}

	fmt.Println(" Done")
	return data
}

func addToRoster(p map[string]string) {
	tempProfile := map[string]string{
		"Last":           p["Last"],
		"First":          p["First"],
		"SunID":          p["SunID"],
		"GoogleID":       p["GoogleID"],
		"GoogleCourseID": p["GoogleCourseID"],
		"CustomID":       p["CustomID"],
		"Email":          p["Email"],
		"GradeLevel":     p["GradeLevel"],
		"Mod":            p["Mod"],
		"Course":         p["Course"],
		"IEP":            p["IEP"],
		"Classroom":      p["Classroom"],
		"Sunguard":       p["Sunguard"],
	}

	c := tempProfile["CustomID"]
	// Check if student is already in map

	// If student is in map, else
	if profile, ok := Roster[c]; ok {
		// compare roster keys to profile keys.  If not in roster keys, add profile key:value to roster
		log.Printf("\n%s %s is already on the roster.  Adding missing information...\n", p["First"], p["Last"])
		for _, key := range keyList {
			// if key from keyList has already been added to the profile
			if val, ok := profile[key]; ok {
				if val == "" {
					if tempProfile[key] != "" {
						log.Printf("%q: %q --> %q\n", key, profile[key], tempProfile[key])
						Roster[c][key] = tempProfile[key]
					}
				} else {
					/* log.Printf("Value %q already stored in key %q\n", val, key) */
					log.Printf("%q: %q --> %q (No Change)\n", key, val, val)
				}

			} else {
				/* log.Printf("\nNo value in either Roster or provided profile for key %q\n", key) */
				log.Printf("%q: %q --> %q (No Change, blank values)\n", key, val, val)
			}
		}
	} else {
		log.Printf("\n%s %s (CustomID = %s) is not on the roster yet.  Adding missing information...\n", tempProfile["First"], tempProfile["Last"], tempProfile["CustomID"])

		Roster[c] = make(map[string]string)
		for _, key := range keyList {
			Roster[c][key] = tempProfile[key]
		}
	}
	// if no, add all info
	// if yes, add info that is missing
}

// PostToSheet takes in a roster (a map of maps) and posts it Row-wise to a specified Google Sheet
func PostToSheet(r map[string]map[string]string) {
	// Post to Google Sheets
	client := gauth.Authorize()

	writeRange := "Master!A2:O"

	// Clear the sheet
	gsheets.Clear(client, SpreadsheetID, writeRange)

	values := make([][]interface{}, len(r))

	// Make a []interface {} and fill with relevant information
	j := 0

	for _, s := range r {

		// Make Last, First and First Last
		lcf := []string{s["Last"], s["First"]}
		lastCommaFirst := strings.Join(lcf, ", ")
		fl := []string{s["First"], s["Last"]}
		firstLast := strings.Join(fl, " ")

		// Construct payload
		if j < len(r) {
			values[j] = []interface{}{s["First"], s["Last"], s["Mod"], s["Course"], s["IEP"], s["GradeLevel"], s["Email"], s["GoogleID"], s["GoogleCourseID"], s["SunID"], s["Sunguard"], s["Classroom"], lastCommaFirst, firstLast, s["CustomID"]}
		}
		j++
	}

	gsheets.BatchUpdateValues(client, SpreadsheetID, writeRange, "ROWS", values)
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
