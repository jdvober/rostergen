package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/jdvober/bak/goClassroomTools/students"
	"github.com/jdvober/gauth"
	"github.com/jdvober/gclass"
	"github.com/jdvober/gsheets"
	"google.golang.org/api/people/v1"
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
	"Last", "First", "SunID", "GoogleID", "GoogleCourseID", "CustomID", "Email", "ParentEmail", "GradeLevel", "Mod", "Course", "IEP", "APEX", "Classroom", "Sunguard", "Contacts", "DateAdded",
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
	apexData := getAPEXData()
	sunguardData := getSunguardData()
	iepData := getIEPData()
	parentEmailData := getParentEmails()

	for _, cd := range classroomData {
		addToRoster(cd)
	}
	for _, ad := range apexData {
		addToRoster(ad)
	}
	for _, sd := range sunguardData {
		addToRoster(sd)
	}
	for _, iepd := range iepData {
		addToRoster(iepd)
	}
	for _, ped := range parentEmailData {
		addToRoster(ped)
	}
	/* updateContacts() */
	/* fmt.Println("Number of students on roster: ", len(Roster))
	 * count := 0
	 * for _, student := range Roster {
	 *     if count < 50 {
	 *         dataJSON := mssToJSON(student)
	 *         fmt.Println(dataJSON)
	 *     }
	 *     count++
	 * } */

	fmt.Printf("\nPosting to sheet...")
	PostToSheet(Roster)
	fmt.Printf(" Done")
}

func getClassroomData() []map[string]string {
	fmt.Printf("\nGetting student profiles from Google Classroom...")

	data := []map[string]string{}

	client := gauth.Authorize()
	courses := gclass.List(client)
	var spreadsheetProfiles []students.Profile

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

func getAPEXData() []map[string]string {
	fmt.Printf("\nGetting student profiles from APEX Paste on Google Sheet...")

	data := []map[string]string{}

	// Get Google Client
	client := gauth.Authorize()
	// Get the pasted in values from Sunguard
	firstNameRange := "APEX Paste!A2:A"
	lastNameRange := "APEX Paste!B2:B"

	firstNames := gsheets.GetValues(client, SpreadsheetID, firstNameRange)
	lastNames := gsheets.GetValues(client, SpreadsheetID, lastNameRange)

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
		/* log.Printf("\nSending %s %s's profile to addToRoster()\n", fn, ln)
		 * addToRoster(profile) */
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
		case "0232-1":
			courses = append(courses, "AP Physics")
			mods = append(mods, "3")
		case "0230-1":
			courses = append(courses, "Physics")
			mods = append(mods, "3")
		case "A0219-1":
			courses = append(courses, "APEX Chemistry")
			mods = append(mods, "3")
		case "A0209-1":
			courses = append(courses, "APEX Physical Science")
			mods = append(mods, "3")
		case "A0230-1":
			courses = append(courses, "APEX Physics")
			mods = append(mods, "3")
		case "A0219H-1":
			courses = append(courses, "APEX Honors Chemistry")
			mods = append(mods, "18")
		case "A0219-2":
			courses = append(courses, "APEX Chemistry")
			mods = append(mods, "24")
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

func getParentEmails() []map[string]string {
	fmt.Printf("\nGetting Parent Emails...")

	// Get Google Client
	client := gauth.Authorize()

	readRange := "Master!J2:R"
	vals := gsheets.GetValues(client, SpreadsheetID, readRange)

	data := []map[string]string{}

	for _, email := range vals {

		parentEmail := email[0].(string)
		cID := email[8].(string)

		profile := map[string]string{
			"CustomID":    cID,
			"ParentEmail": parentEmail,
		}
		data = append(data, profile)
	}

	fmt.Println(" Done")
	return data
}

func updateContacts() {
	fmt.Printf("\nUpdating Google Contacts for email lists...")

	client := gauth.Authorize()
	srv, err := people.New(client)
	if err != nil {
		log.Fatalf("Unable to create people Client %v", err)
	}

	/* ctx := context.Background()
	 * peopleService, err := people.NewService(ctx) */

	/* contactGroups := map[string]string{
	 *     "APEX":                         "contactGroups/173aa8400bbdd3b1",
	 *     "APEXChemistry":                "contactGroups/2ae069080f05aacf",
	 *     "APEXPhysicalScience":          "contactGroups/220329b90ad8cae7",
	 *     "APEXPhysics":                  "contactGroups/6a81e9bc0a416f52",
	 *     "APEXHonorsChemistry2020_2021": "contactGroups/6e054fb98b8ec269",
	 * } */

	// Compare Master to each contactGroup the student should be a part of (based on class, etc.)

	cIDsMaster := gsheets.GetValues(client, SpreadsheetID, "Master!R2:R")
	classes := gsheets.GetValues(client, SpreadsheetID, "Master!D2:D")
	firstNames := gsheets.GetValues(client, SpreadsheetID, "Master!A2:A")
	lastNames := gsheets.GetValues(client, SpreadsheetID, "Master!B2:B")
	spreadsheetProfiles := map[string]map[string]string{}
	contactsProfiles := map[string]map[string]string{}

	// Make initial profiles from master list so you have something to compare the values returned from Contacts to
Loop:
	for c, mID := range cIDsMaster {
		var cID string
		var class string
		var first string
		var last string

		if len(classes[c]) > 0 {
			class = classes[c][0].(string)
			cID = mID[0].(string)
			first = firstNames[c][0].(string)
			last = lastNames[c][0].(string)

		} else {
			cID = mID[0].(string)
			continue Loop
		}

		spreadsheetProfiles[cID] = map[string]string{
			"id":    cID,
			"class": class,
			"first": first,
			"last":  last,
			"full":  first + " " + last,
		}

		fmt.Printf("\n%+v", spreadsheetProfiles[cID])
	}

	// Get all students that are currently in my contacts, with their names and contactGroups

	r, err := srv.People.Connections.List("people/me").PageSize(1000).
		PersonFields("names,memberships").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve people. %v", err)
	}
	fmt.Printf("\nThere are currently %d people on your contacts list\n", len(r.Connections))
	for _, connection := range r.Connections {
		// Construct customID format lastFIRST
		if len(connection.Names) > 0 {
			fmt.Printf("\n%s is listed as a Google Contact\n", connection.Names[0].DisplayName)
			cID := strings.ToLower(connection.Names[0].FamilyName) + strings.ToUpper(connection.Names[0].GivenName)

			contactGroupResourceName := connection.Memberships[1].ContactGroupMembership.ContactGroupResourceName
			classFromContacts, err := srv.ContactGroups.Get(contactGroupResourceName).Do()
			if err != nil {
				log.Printf("Error getting name of list student is in on Google Contacts")
			}
			contactsProfiles[cID] = map[string]string{
				"id":           cID,
				"displayName":  connection.Names[0].DisplayName,
				"first":        connection.Names[0].GivenName,
				"last":         connection.Names[0].FamilyName,
				"resourceName": connection.ResourceName,
				"classID":      connection.Memberships[1].ContactGroupMembership.ContactGroupResourceName,
				"class":        classFromContacts.Name,
			}
			/*             fmt.Printf("\nAssigning class for %s\n", mssToJSON(contactsProfiles[cID]))
			 *             for _, cl := range connection.Memberships {
			 *                 fmt.Printf("\nSwitch %s\n", cl.ContactGroupMembership.ContactGroupResourceName)
			 *                 switch cl.ContactGroupMembership.ContactGroupResourceName {
			 *                 case "contactGroups/173aa8400bbdd3b1":
			 *                     contactsProfiles[cID]["APEX"] = "contactGroups/173aa8400bbdd3b1"
			 *                     fallthrough
			 *                 case "contactGroups/2ae069080f05aacf":
			 *                     contactsProfiles[cID]["classID"] = "contactGroups/2ae069080f05aacf"
			 *                     contactsProfiles[cID]["class"] = "APEX Chemistry"
			 *                     fallthrough
			 *                 case "contactGroups/6e054fb98b8ec269":
			 *                     contactsProfiles[cID]["classID"] = "contactGroups/6e054fb98b8ec269"
			 *                     contactsProfiles[cID]["class"] = "APEX Honors Chemistry"
			 *                     fallthrough
			 *                 case "contactGroups/6a81e9bc0a416f52":
			 *                     contactsProfiles[cID]["classID"] = "contactGroups/6a81e9bc0a416f52"
			 *                     contactsProfiles[cID]["class"] = "APEX Physics"
			 *                     fallthrough
			 *                 case "contactGroups/220329b90ad8cae7":
			 *                     contactsProfiles[cID]["classID"] = "contactGroups/220329b90ad8cae7"
			 *                     contactsProfiles[cID]["class"] = "APEX Physical Science"
			 *                 default:
			 *                     contactsProfiles[cID]["classID"] = ""
			 *                     contactsProfiles[cID]["class"] = ""
			 *
			 *                 } */
			/* } */

		}
	}

	for _, ssProfile := range spreadsheetProfiles {
		for _, contactsProfile := range contactsProfiles {
			fmt.Printf("\n\nDoes ssProfile['id']: %s match contactsProfile['id']: %s?\n", ssProfile["id"], contactsProfile["id"])
			if ssProfile["id"] == contactsProfile["id"] {
				// FOUND A MATCH!
				fmt.Printf("YES!  They match!\n")
				fmt.Printf("ssProfile['class']: %s\tcontactsProfile['class']: %s\n", ssProfile["class"], contactsProfile["class"])
				// Check what classes they are in.  If it matches, good, else update
				switch contactsProfile["class"] {
				case "APEX Chemistry", "APEX Honors Chemistry", "APEX Physics", "APEX Physical Science":
					if ssProfile["class"] == contactsProfile["class"] {
						fmt.Printf("%s is listed as a member of %q on both the Spreadsheet and in Google Contacts.\n", ssProfile["full"], ssProfile["class"])
					} else {
						// Classes do not match.  Add to Correct contacts list and remove from other list IF NOT AP Physics or Physics!!!
						fmt.Printf("A match was found for %s %s, but their listed classes do not match up!\n", contactsProfile["First"], contactsProfile["Last"])
						// addToContacts(resourceNamePerson, resourceNameClass)
						// removeFromContacts(resourceNamePerson, resourceNameClass)
					}
				default:
					fmt.Printf("\nSomething went wrong.")
				}
			} else {
				// No Match Found, add to contacts
				fmt.Printf("%s %s not found in Contacts.  Adding to proper classes...\n", ssProfile["first"], ssProfile["last"])
				// Get resourceName of student and resourceName of Class you want to add them to in Google Contacts

				res, err := srv.People.ListDirectoryPeople().MergeSources("DIRECTORY_MERGE_SOURCE_TYPE_CONTACT").ReadMask("names").Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").Do()
				if err != nil {
					log.Printf("%s\n", err)
				}
				fmt.Printf("\nres = \n%v", res)

			}
		}
	}

	// Check to see if they are in APEX Group

	// Check to see if they are in their required class group

	// Make a customID for the name returned from the contactGroup.
	// If they are found, determine if they should be in that group.
	// If yes, all good.
	// If no, remove them.
	// If they are NOT found, determine if they SHOULD be in that group.
	// If yes, add them.
	// If no, add good.

	/*
		r, err := srv.People.Connections.List("people/me").PageSize(1000).
			PersonFields("names,emailAddresses").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve people. %v", err)
		}
		if len(r.Connections) > 0 {
			fmt.Print("\nListing first 1000 connection names:\n")
			for _, c := range r.Connections {

				names := c.Names
				resourceName := c.ResourceName
				if len(names) > 0 {
					name := names[0].DisplayName

					var rb people.ModifyContactGroupMembersRequest = people.ModifyContactGroupMembersRequest{
						ResourceNamesToAdd: []string{resourceName},
					}

					rb.MarshalJSON()
					fmt.Printf("\nAdding %s (resourceName: %q) to contact group %s", name, resourceName, contactGroup)
					// Add to Necessary Contact Groups
					srv.ContactGroups.Members.Modify(contactGroup, &rb).Do()
					if err != nil {
						log.Println("An error occured when attempting to modify the Google Contacts Group %q with resourceName %q\n", contactGroup, resourceName)
					}
				}
			}
		} else {
			fmt.Print("No connections found.")
		}
	*/

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
		"ParentEmail":    p["ParentEmail"],
		"GradeLevel":     p["GradeLevel"],
		"Mod":            p["Mod"],
		"Course":         p["Course"],
		"IEP":            p["IEP"],
		"APEX":           p["APEX"],
		"Classroom":      p["Classroom"],
		"Sunguard":       p["Sunguard"],
		"Contacts":       p["Contacts"],
		"DateAdded":      p["DateAdded"],
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
func addToContacts(resourceNamePerson string, resourceNameContactGroup string) {
	client := gauth.Authorize()
	srv, err := people.New(client)
	if err != nil {
		log.Fatalf("Unable to create people Client %v", err)
	}

	var rb people.ModifyContactGroupMembersRequest = people.ModifyContactGroupMembersRequest{
		ResourceNamesToAdd: []string{resourceNamePerson},
	}

	rb.MarshalJSON()
	fmt.Printf("\nAdding resourceName: %q to contact group %s", resourceNamePerson, resourceNameContactGroup)
	// Add to Necessary Contact Groups
	srv.ContactGroups.Members.Modify(resourceNameContactGroup, &rb).Do()
	if err != nil {
		log.Printf("An error occured when attempting to modify the Google Contacts Group %q with resourceName %q\n", resourceNameContactGroup, resourceNamePerson)
	}
}

// PostToSheet takes in a roster (a map of maps) and posts it Row-wise to a specified Google Sheet
func PostToSheet(r map[string]map[string]string) {
	// Post to Google Sheets
	client := gauth.Authorize()

	writeRange := "Master!A2:S"

	// Save old customIDs and Dates for comparison later
	readRange := "Master!R2:S"
	oldIDsAndDates := gsheets.GetValues(client, SpreadsheetID, readRange)
	/* savedParentEmails := gsheets.GetValues(client, SpreadsheetID, "Master!J2:J") */

	// Clear the sheet
	gsheets.Clear(client, SpreadsheetID, writeRange, "ROWS")

	values := make([][]interface{}, len(r))

	// Make a []interface {} and fill with relevant information
	j := 0
	ct := time.Now()
	currentTime := ct.Format("01.02.2006 15:04:05")

	for _, s := range r {
		// If s["CustomID"] is in oldIDsAndDates
		for _, i := range oldIDsAndDates {
			if i[0].(string) == s["CustomID"] {
				currentTime = i[1].(string)
				break
			}
		}

		// Make Last, First and First Last
		lcf := []string{s["Last"], s["First"]}
		lastCommaFirst := strings.Join(lcf, ", ")
		fl := []string{s["First"], s["Last"]}
		firstLast := strings.Join(fl, " ")

		// Construct payload
		if j < len(r) {
			values[j] = []interface{}{s["First"], s["Last"], s["Email"], s["Course"], s["GradeLevel"], s["Mod"], s["GoogleID"], s["GoogleCourseID"], s["SunID"], s["ParentEmail"], s["IEP"], s["Sunguard"], s["APEX"], s["Classroom"], s["Contacts"], lastCommaFirst, firstLast, s["CustomID"], currentTime}
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
