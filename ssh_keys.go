package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// now returns empty passphrase
func AskPass(prompt string) []byte {
	// os.Stderr.Write([]byte("\n" + prompt + ": "))
	return []byte("")
}

func LoadPrivateKey() ssh.Signer {
	id_rsa := FindSshPvtKeyFile("")
	if id_rsa == "" {
		log.Warn("No private key file here")
		return nil
	}
	pem, err := ioutil.ReadFile(id_rsa)
	if err != nil {
		log.Fatal("Cannot read %q: %v", id_rsa, err)
	}
	sgn, err := ssh.ParsePrivateKey(pem)
	if err == nil {
		return sgn
	}
	switch err.(type) {
	case *ssh.PassphraseMissingError:
		pass := AskPass(id_rsa)
		sgn, err = ssh.ParsePrivateKeyWithPassphrase(pem, pass)
		if err == nil {
			return sgn
		}
		return nil
	}
	log.Fatal("Cannot parse %q: %v", id_rsa, err)
	return nil
}

func LoadPublicKey() ssh.PublicKey {
	id_rsa := FindSshPubKeyFile("")
	if id_rsa == "" {
		log.Warn("No public key file here")
		return nil
	}
	pem, err := ioutil.ReadFile(id_rsa)
	if err != nil {
		log.Fatal("Cannot read %q: %v", id_rsa, err)
	}
	pub, err := ssh.ParsePublicKey(pem)
	if err != nil {
		log.Fatal("Cannot parse %q: %v", id_rsa, err)
	}
	return pub
}

func sha1hmac(salt, text string) string {
	mac := hmac.New(sha1.New, []byte(salt))
	mac.Write([]byte(text))
	return string(mac.Sum(nil))
}

func unbase64(text string) string {
	res, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		log.Fatal("Cannot unbase64 %q: %v", text, err)
	}
	return string(res)
}

func IsHostKeyEntry(host, entry string) bool {
	if !strings.HasPrefix(entry, "|") {
		return host == entry
	}
	fx := strings.Split(entry, "|")
	if len(fx[0]) != 0 {
		log.Fatal("Bad format %q", entry)
	}
	switch fx[1] {
	case "1": // |1|<salt>|<hash>|... // https://habr.com/en/post/421477/
		salt, hash := unbase64(fx[2]), unbase64(fx[3])
		return hash == sha1hmac(salt, host)
	default:
		log.Fatal("Unknown hash %q in %q", fx[1], entry)
	}
	return false // never reached
}

func FindHostKey(path, host string) ssh.PublicKey {
	// https://github.com/Nokta-strigo/known_hosts_parser
	kh, err := os.Open(path)
	if err != nil {
		log.Fatal("Cannot open %q: %v", path, err)
	}
	defer kh.Close()

	scanner := bufio.NewScanner(kh)
	ln := 0
	for scanner.Scan() {
		ln += 1
		line := scanner.Text()
		if IsCommentOrBlank(line) {
			continue
		}
		if strings.HasPrefix(line, "@") {
			word := strings.Split(line, " ")
			log.Debug("%3d: marker %q", ln, word[0])
			line = strings.Join(word[1:], " ")
		}
		word := strings.Split(line, " ")
		cmnt := ""
		if len(word) > 3 {
			cmnt = " [" + strings.Join(word[3:], " ") + "]"
		}
		if IsHostKeyEntry(host, word[0]) {
			hostKey, _, _, _, err := ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				log.Fatal("Error parsing %v: %v", word, err)
			}
			log.Debug("Found %q key for host %q at line %d%s",
				word[1], host, ln, cmnt)
			return hostKey
		}
	}
	return nil
}

/* EOF */
