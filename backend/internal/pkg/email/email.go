// Package email отправляет простые текстовые письма через SMTP (в dev - на
// Mailhog из docker-compose, не требующий авторизации; в проде - через
// SMTPHost/Username/Password, если/когда они будут сконфигурированы отдельно
// от текущего hardcoded-на-mailhog docker-compose.yml).
package email

import (
	"fmt"
	"net/smtp"

	"github.com/medflow/backend/internal/config"
)

type Sender struct {
	host, port, username, password, from string
}

func NewSender(cfg config.EmailConfig) *Sender {
	return &Sender{host: cfg.SMTPHost, port: cfg.SMTPPort, username: cfg.Username, password: cfg.Password, from: cfg.From}
}

// Send отправляет простое текстовое письмо. Auth пропускается, если
// Username пуст - Mailhog (и большинство локальных relay) авторизации не
// требует; реальный прод-провайдер потребует Username/Password заполненными.
func (s *Sender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
