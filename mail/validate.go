package mail

import (
	"net"
	"net/mail"
	"strings"
)

// Validate performs some basic email address validation on a given
// address, just ensuring it's indeed a valid address according to
// RFC 5322. If useNetwork is true, the domain will be also validated.
// Even if this function returns no error, IT DOESN'T MEAN THE
// ADDRESS EXISTS. The only way to be completely sure the address
// exist and can receive email is sending an email with a link back
// to your site including a randomly generated token that the user
// has to click to verify the he can read email sent to that address.
func Validate(address string, useNetwork bool) error {
	addr, err := mail.ParseAddress(address)
	if err != nil {
		return err
	}
	if useNetwork {
		host := strings.Split(addr.Address, "@")[1]
		mx, err := net.LookupMX(host)
		if err == nil {
			for _, v := range mx {
				if _, err := net.LookupHost(v.Host); err == nil {
					// We have a valid MX
					return nil
				}
			}
		}
		// Try a A lookup
		_, err = net.LookupHost(host)
		if err != nil {
			return err
		}
	}
	return nil
}
