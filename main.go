package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/minio/minio-go/v7"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

type ConnectionInformation struct {
	Name                  string `yaml:"name"`
	Endpoint              string `yaml:"endpoint"`
	AccessKey             string `yaml:"access_key"`
	SecretKey             string `yaml:"secret_key"`
	IgnoreSSLVerification bool   `yaml:"ignore_ssl_verification"`
}

var app *tview.Application
var minioClient *minio.Client
var connectionList *tview.List
var fileExplorer *tview.TreeView

// Display feedback in a modal
func showModal(message string, returnToForm bool, form *tview.Form) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if returnToForm && form != nil {
				app.SetRoot(form, true)
			} else {
				app.SetRoot(mainLayout(), true)
			}
		})
	app.SetRoot(modal, true)
}

// Function to add child nodes to the TreeNode for S3 objects
func addObjectsToTree(target *tview.TreeNode, bucketName, prefix string) {
	ctx := context.Background()

	// List objects in the S3 bucket
	objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false, // List non-recursively for folders
	})

	for object := range objectCh {
		if object.Err != nil {
			log.Printf("Failed to list objects: %v", object.Err)
			return
		}

		// Determine if the object is a folder or file
		node := tview.NewTreeNode(strings.TrimPrefix(object.Key, prefix)).
			SetReference(object.Key).
			SetSelectable(true)

		if strings.HasSuffix(object.Key, "/") {
			// It's a folder (prefix)
			node.SetColor(tcell.ColorGreen)
		} else {
			// It's a file
			node.SetColor(tcell.ColorLavender)
		}

		// Add the node directly to the target
		target.AddChild(node)
	}
}

// Function to display the file explorer within the right panel
func showFileExplorer(bucketName string) {
	// Set up root node for the selected bucket
	root := tview.NewTreeNode(bucketName).SetColor(tcell.ColorLavender)
	fileExplorer.SetRoot(root).SetCurrentNode(root)

	// Load root-level objects in the bucket
	addObjectsToTree(root, bucketName, "")

	// Expand folders and load children when selected
	fileExplorer.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			return
		}

		// If this is a folder (S3 prefix), load its contents if not loaded yet
		if strings.HasSuffix(ref.(string), "/") {
			children := node.GetChildren()
			if len(children) == 0 {
				addObjectsToTree(node, bucketName, ref.(string))
			}
			node.SetExpanded(!node.IsExpanded())
		}
	})
}

// Form for creating a new connection
func showConnectionForm() {
	form := tview.NewForm()
	var info ConnectionInformation

	form.
		AddInputField("Name", "", 30, nil, func(text string) {
			info.Name = text
		}).
		AddInputField("S3 Endpoint", "", 30, nil, func(text string) {
			info.Endpoint = text
		}).
		AddInputField("Access Key", "", 30, nil, func(text string) {
			info.AccessKey = text
		}).
		AddPasswordField("Secret Key", "", 30, '*', func(text string) {
			info.SecretKey = text
		}).
		AddCheckbox("Ignore SSL Verification", false, func(checked bool) {
			info.IgnoreSSLVerification = checked
		}).
		AddButton("Save", func() {
			if strings.TrimSpace(info.Name) == "" || strings.TrimSpace(info.Endpoint) == "" || strings.TrimSpace(info.AccessKey) == "" || strings.TrimSpace(info.SecretKey) == "" {
				showModal("All fields are required.", true, form)
				return
			}
			err := saveConnection(info)
			if err != nil {
				showModal(fmt.Sprintf("Failed to save connection: %v", err), true, form)
				return
			}
			showModal("Connection information saved successfully!", false, nil)
		}).
		AddButton("Cancel", func() {
			app.SetRoot(mainLayout(), true)
		})

	form.SetBorder(true).SetTitle("New Connection").SetTitleAlign(tview.AlignCenter)
	app.SetRoot(form, true)
}

func mainLayout() *tview.Flex {
	connectionList = tview.NewList()
	connectionList.SetBorder(true).SetTitle("Connections (Press C to Connect)")

	for _, file := range loadConnections() {
		connectionList.AddItem(file, "", 'c', func(filename string) func() {
			return func() {
				content, err := os.ReadFile(filename)
				if err != nil {
					showModal(fmt.Sprintf("Failed to read connection file: %v", err), false, nil)
					return
				}
				var info ConnectionInformation
				err = yaml.Unmarshal(content, &info)
				if err != nil {
					showModal(fmt.Sprintf("Failed to parse connection file: %v", err), false, nil)
					return
				}
				if err := connectToS3(info); err != nil {
					showModal(fmt.Sprintf("Failed to connect: %v", err), false, nil)
					return
				}

				// Retrieve buckets and display the first bucket in file explorer
				buckets, err := minioClient.ListBuckets(context.Background())
				if err != nil || len(buckets) == 0 {
					showModal("Failed to retrieve buckets or no buckets found.", false, nil)
					return
				}

				// Pass the correct bucket name to showFileExplorer
				showFileExplorer(buckets[0].Name)
			}
		}(file))
	}

	fileExplorer = tview.NewTreeView()
	fileExplorer.SetBorder(true).SetTitle("File Explorer")

	buttonsTable := tview.NewTable().SetBorders(true)
	buttonsTable.SetCell(0, 0, tview.NewTableCell("Quit").SetAlign(tview.AlignCenter).SetSelectable(false))
	buttonsTable.SetCell(0, 1, tview.NewTableCell("Add Connection").SetAlign(tview.AlignCenter).SetSelectable(false))
	// quitButton.SetBackgroundColor(tcell.ColorDefault)
	// newConnectionButton := tview.NewButton("Add Connection").SetSelectedFunc(showConnectionForm)
	buttons := tview.NewFlex().
		AddItem(buttonsTable, 0, 1, false) // Only text is green

	// Combine connection list and file explorer in a horizontal flex
	contentFlex := tview.NewFlex().
		AddItem(connectionList, 0, 1, true).
		AddItem(fileExplorer, 0, 2, false)

	// Main layout with content and buttons at the bottom
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(contentFlex, 0, 1, true). // Main content
		AddItem(buttons, 3, 1, false)     // Buttons at the bottom

	// Input capture to navigate between panels
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() == tcell.ModAlt {
			switch event.Key() {
			case tcell.KeyLeft:
				app.SetFocus(connectionList) // Focus on the connection list
				return nil
			case tcell.KeyRight:
				app.SetFocus(fileExplorer) // Focus on the file explorer
				return nil
			case tcell.KeyDown:
				app.SetFocus(buttons) // Focus on the buttons area
				return nil
			case tcell.KeyUp:
				app.SetFocus(contentFlex) // Move back to main content
				return nil
			}
		}
		return event
	})

	return mainFlex
}

func main() {
	app = tview.NewApplication()
	if err := app.SetRoot(mainLayout(), true).EnableMouse(true).Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
