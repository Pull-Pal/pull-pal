package main

import "github.com/mobyvb/pull-pal/cmd"

func main() {
	/*
		response := "Files:\n\n  - name: main.go\n    contents:\n      ```go\n      package main\n\n      import (\n        \"fmt\"\n        \"log\"\n        \"net/http\"\n      )\n\n      func main() {\n        http.Handle(\"/\", http.FileServer(http.Dir(\"./\")))\n        fmt.Println(\"Server listening on :7777\")\n        log.Fatal(http.ListenAndServe(\":7777\", nil))\n      }\n      ```\n\n  - name: index.html\n    contents:\n      ```html\n      <!DOCTYPE html>\n      <html>\n      <head>\n        <title>Pull Pal</title>\n        <style>\n          body {\n            font-family: sans-serif;\n          }\n          .content {\n            background-color: #f2f2f2;\n            border-radius: 10px;\n            box-shadow: 2px 2px 10px rgba(0,0,0,0.2);\n            padding: 20px;\n            margin: 20px;\n          }\n          h1 {\n            color: #3399cc;\n            background-color: #f2f2f2;\n            text-align: center;\n            padding: 10px;\n            border-radius: 10px;\n            box-shadow: 2px 2px 10px rgba(0,0,0,0.2);\n          }\n        </style>\n      </head>\n      <body>\n        <div class=\"content\">\n          <h1>Introducing Pull Pal!</h1>\n          <p>Pull Pal is a digital assistant that can monitor your Github repositories and create pull requests using the power of artificial intelligence. Say goodbye to manual pull request creation like it's 2005!</p>\n          <p>Sign up now to start automating your workflow and get more done in less time. </p>\n        </div>\n      </body>\n      </html>\n      ```\n\nNotes:\n- Added a basic HTTP server in main.go which serves index.html from the root path on port 7777\n- Added basic HTML structure and CSS styling to index.html, using sans-serif font and a soft color scheme with blue as the main accent color\n- Added a content container with an off-white background, rounded corners, and a box shadow, and a centered heading with blue font color and an off-white background to draw attention to the product name and value proposition"
		res := llm.ParseCodeChangeResponse(response)
		fmt.Println(res.String())
	*/
	cmd.Execute()
}
