package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var stdinReader = bufio.NewReader(os.Stdin)

func promptString(label string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue, nil
	}
	return text, nil
}

func promptRequired(label string, defaultValue string) (string, error) {
	value, err := promptString(label, defaultValue)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return value, nil
}

func promptPassword(label string) (string, error) {
	fmt.Printf("%s: ", label)
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(passwordBytes)), nil
}

func promptConfirm(label string, defaultYes bool) (bool, error) {
	suffix := "y/N"
	if defaultYes {
		suffix = "Y/n"
	}
	value, err := promptString(fmt.Sprintf("%s [%s]", label, suffix), "")
	if err != nil {
		return false, err
	}
	if value == "" {
		return defaultYes, nil
	}
	switch strings.ToLower(value) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, errors.New("please answer yes or no")
	}
}

func promptSelect(label string, options []string) (string, error) {
	if len(options) == 0 {
		return "", errors.New("no options available")
	}
	fmt.Println(label)
	for index, option := range options {
		fmt.Printf("  %d. %s\n", index+1, option)
	}
	choice, err := promptRequired("Select a number", "")
	if err != nil {
		return "", err
	}
	number, err := strconv.Atoi(choice)
	if err != nil || number < 1 || number > len(options) {
		return "", errors.New("invalid selection")
	}
	return options[number-1], nil
}
