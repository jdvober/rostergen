module goRoster

go 1.15

//Comment these lines out when using for production
//////////////////////////////////////////////////////////

replace github.com/jdvober/goSheets/values => ../goSheets/values/

//
replace github.com/jdvober/goClassroomTools/students => ../goClassroomTools/students

//
replace github.com/jdvober/goClassroomTools/courses => ../goClassroomTools/courses

//////////////////////////////////////////////////////////

require (
	github.com/jdvober/goClassroomTools/courses v0.0.0-00010101000000-000000000000
	github.com/jdvober/goClassroomTools/students v0.0.0-00010101000000-000000000000
	github.com/jdvober/goGoogleAuth v0.0.0-20201015191935-8a1c594381c2
	github.com/jdvober/goSheets/values v0.0.0-00010101000000-000000000000
)
