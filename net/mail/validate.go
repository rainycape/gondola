package mail

import (
	"net"
	"net/mail"
	"strings"

	"gnd.la/internal"
)

// Validate performs some basic email address validation on a given
// address, just ensuring it's indeed a valid address according to
// RFC 5322. If useNetwork is true, the domain will be also validated.
// Even if this function returns no error, IT DOESN'T MEAN THE
// ADDRESS EXISTS. The only way to be completely sure the address
// exist and can receive email is sending an email with a link back
// to your site including a randomly generated token that the user
// has to click to verify the he can read email sent to that address.
// The returned string is the address part of the given string (e.g.
// "Alberto G. Hierro <alberto@garciahierro.com>" would return
// "alberto@garciahierro").
//
// Note for GAE: Due to the GAE runtime restrictions, there's no way to
// perform DNS lookus, so the useNetwork parameter is ignored when running
// on GAE.
func Validate(address string, useNetwork bool) (email string, err error) {
	var addr *mail.Address
	addr, err = mail.ParseAddress(address)
	if err != nil {
		return
	}
	if useNetwork && !internal.InAppEngine() {
		// App Engine does not provide any way to check DNS records.
		// For now, always return true. TODO: Find a better solution
		err = validateNetworkAddress(addr.Address)
	}
	if err == nil {
		email = addr.Address
	}
	return
}

func validateNetworkAddress(address string) error {
	host := strings.Split(address, "@")[1]
	mx, err := net.LookupMX(host)
	if err == nil {
		for _, v := range mx {
			if _, err := net.LookupHost(v.Host); err == nil {
				// We have a valid MX
				return nil
			}
		}
	}
	// Try an A lookup
	_, err = net.LookupHost(host)
	if err != nil {
		return err
	}
	return nil
}
