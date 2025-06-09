# Project Context for AI Assistance

never generate xml from hard coded string templates with %s placeholders, use structs
Always test for DTD validation after changing code. 

xmllint --dtdvalid FCPXMLv1_13.dtd output.fcpxml

this program is a swiff army knife for generating fcpxml files. There is a complex cli menu system for asking what specific army knife you want.

if you are confident in what you just changed, don't test it. Just make sure it compiles.

do not add complex logic to main.go that belongs in other packages.
have main.go call funcs in a package instead.
