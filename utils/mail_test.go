// Copyright 2019 asana Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import "testing"

func TestMail(t *testing.T) {
	config := `{"username":"asana@gmail.com","password":"asana","host":"smtp.gmail.com","port":587}`
	mail := NewEMail(config)
	if mail.Username != "asana@gmail.com" {
		t.Fatal("email parse get username error")
	}
	if mail.Password != "asana" {
		t.Fatal("email parse get password error")
	}
	if mail.Host != "smtp.gmail.com" {
		t.Fatal("email parse get host error")
	}
	if mail.Port != 587 {
		t.Fatal("email parse get port error")
	}
	mail.To = []string{"xiemengjun@gmail.com"}
	mail.From = "asana@gmail.com"
	mail.Subject = "hi, just from asana!"
	mail.Text = "Text Body is, of course, supported!"
	mail.HTML = "<h1>Fancy Html is supported, too!</h1>"
	_, _ = mail.AttachFile("/Users/asana/github/asana/asana.go")
	_ = mail.Send()
}
