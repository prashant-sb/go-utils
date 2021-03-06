package users

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"

	log "github.com/golang/glog"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	userDB    string = "/etc/passwd" // User parser file in linux
	userShell string = "/bin/bash"   // Default user shell
	userAdd   string = "useradd"     // Command for adding user
	userDel   string = "userdel"     // Command for deleting user
)

type Userinfo struct {
	// Uid is the user ID.
	Uid string `json:"uid"`

	// Gid is the primary group ID.
	Gid string `json:"gid"`

	// Username is the login name.
	Username string `json:"userName,omitempty"`

	// Group name / optional
	Groupname string `json:"groupName,omitempty"`

	// Name is the user's real or display name.
	// It might be blank.
	Name string `json:"name,omitempty"`

	// HomeDir is the path to the user's home directory
	// (if they have one).
	HomeDir string `json:"homeDir,omitempty"`
	// Added for unit tests

	UserPasswd string `json:"userPasswd,omitempty"`
}

type UserList struct {
	// Userinfo lists for all system users
	Users []Userinfo `json:"users"`
}

type UserOps interface {

	// Exported methods for Userinfo
	Get(string) (*Userinfo, error)
	AddUser(string) (string, error)
	DeleteUser(string) (string, error)

	// Private methods for Userinfo
	add(*Userinfo) error
	delete(*Userinfo) error
	creadential() (string, error)
	readUsers(string) ([]byte, error)
}

// UserListOps interface provides
// list of all system users by parsing
// Linux password file.
type UserListOps interface {
	Get() (*UserList, error)
	ReadEtcPasswd(string) ([]string, error)
}

// NewUserOps inits the interface for Userinfo
func NewUserOps() UserOps {
	return &Userinfo{}
}

// NewUserList inits the interface for UserList
func NewUserList() UserListOps {
	return &UserList{
		Users: []Userinfo{},
	}
}

// Functions that binds to UserList interface

// Get the userlist of all users
func (ul *UserList) Get() (*UserList, error) {

	var userlist []Userinfo

	ulist, err := ul.ReadEtcPasswd(userDB)
	if err != nil {
		return nil, err
	}

	for i := range ulist {
		u, err := user.Lookup(ulist[i])
		if err != nil {
			return nil, err
		}
		g, err := user.LookupGroupId(u.Gid)
		if err != nil {
			return nil, err
		}

		uinfo := Userinfo{
			Uid:       u.Uid,
			Gid:       u.Gid,
			Name:      u.Name,
			HomeDir:   u.HomeDir,
			Username:  u.Username,
			Groupname: g.Name,
		}
		userlist = append(userlist, uinfo)
	}

	return &UserList{
		Users: userlist,
	}, nil
}

// Read file /etc/passwd and return slice of users
func (ul *UserList) ReadEtcPasswd(f string) ([]string, error) {
	var ulist []string

	file, err := os.Open(f)
	if err != nil {
		return ulist, err
	}
	defer file.Close()

	r := bufio.NewScanner(file)

	for r.Scan() {
		lines := r.Text()
		parts := strings.Split(lines, ":")
		ulist = append(ulist, parts[0])
	}
	return ulist, nil
}

// Functions that binds to Userinfo interface

// Get user schema with username
func (u *Userinfo) Get(userName string) (*Userinfo, error) {

	ui, err := user.Lookup(userName)
	if err != nil {
		return nil, err
	}
	g, err := user.LookupGroupId(ui.Gid)
	if err != nil {
		return nil, err
	}

	return &Userinfo{
		Uid:       ui.Uid,
		Gid:       ui.Gid,
		Name:      ui.Name,
		HomeDir:   ui.HomeDir,
		Username:  ui.Username,
		Groupname: g.Name,
	}, nil
}

// AddUser adds the system user with provided schema
func (u *Userinfo) AddUser(usrJsonFile string) (string, error) {

	var usr string
	uinfo := Userinfo{}

	b, err := u.readUsers(usrJsonFile)
	if err != nil {
		log.Error("Error in reading ", usrJsonFile)
		return usr, err
	}

	err = json.Unmarshal(b, &uinfo)
	if err != nil {
		log.Error("Error in unmarshal: ", err.Error())
		return usr, err
	}

	if err = u.add(&uinfo); err != nil {
		log.Error("Error in adding user ", uinfo.Username)
		return "", err
	}

	return uinfo.Username, nil
}

// DeleteUser gets the schema for user by name, deletes it if available.
func (u *Userinfo) DeleteUser(userName string) (string, error) {

	uinfo, err := u.Get(userName)
	if err != nil {
		return "", errors.New("User " + userName + " not found.")
	}

	if err := u.delete(uinfo); err != nil {
		return "", err
	}

	return uinfo.Username, nil
}

// Get the password from stdin for user
func (u *Userinfo) creadential() (string, error) {

	fmt.Printf("Enter Password for %s: ", u.Username)
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	password := string(bytePassword)
	fmt.Println()

	return strings.TrimSpace(password), nil
}

// add user from Userinfo, if new user
func (u *Userinfo) add(uinfo *Userinfo) error {

	var passwd string
	var err error

	if _, err := u.Get(uinfo.Username); err == nil {
		return errors.New("User " + uinfo.Username + " already added.")
	}

	u.Username = uinfo.Username

	if uinfo.UserPasswd != "" {
		passwd = uinfo.UserPasswd
	} else {
		passwd, err = u.creadential()
		if err != nil {
			return err
		}
	}
	argUser := []string{"-m", "-d", uinfo.HomeDir, "-G", uinfo.Groupname, "-s", userShell, uinfo.Username, "-p", passwd}
	userCmd := exec.Command(userAdd, argUser...)

	if _, err := userCmd.Output(); err != nil {
		log.Error("Error in adding user : ", u.Username, " ", err.Error())
		return err
	}

	return nil
}

// deletes provided Userinfo from system
func (u *Userinfo) delete(uinfo *Userinfo) error {

	argUser := []string{"-r", uinfo.Username}
	userCmd := exec.Command(userDel, argUser...)

	if _, err := userCmd.Output(); err != nil {
		log.Error("Error in deleting user : ", uinfo.Username, "-", err.Error())
		return err
	}

	return nil
}

// Read json file and return slice of byte.
func (u *Userinfo) readUsers(f string) ([]byte, error) {

	jsonFile, err := os.Open(f)

	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	data, _ := ioutil.ReadAll(jsonFile)
	return data, nil
}

// Decodes the json from embeded interface to indented string
func Decode(e interface{}) (string, error) {

	var usersJson []byte

	usersJson, err := json.MarshalIndent(e, "", "   ")
	if err != nil {
		return string(usersJson), err
	}

	return string(usersJson), nil
}
