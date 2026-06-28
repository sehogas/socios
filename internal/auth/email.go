package auth

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

// SendMail envía un correo electrónico utilizando SMTP o simulándolo en consola si no está configurado.
func SendMail(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")

	if host == "" {
		// Log de desarrollo local para simular envío de correos
		log.Printf("\n======================================================\n"+
			"[DESARROLLO - CORREO SIMULADO]\n"+
			"Para: %s\n"+
			"Asunto: %s\n"+
			"Mensaje:\n%s\n"+
			"======================================================\n", to, subject, body)
		return nil
	}

	addr := host + ":" + port
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", user, pass, host)
	err := smtp.SendMail(addr, auth, from, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("error al enviar correo por SMTP: %w", err)
	}
	return nil
}

// SendVerificationEmail envía el correo de verificación con el enlace correspondiente.
func SendVerificationEmail(baseURL, toEmail, token string) error {
	link := fmt.Sprintf("%s/verificar-email?token=%s", baseURL, token)
	subject := "Verifica tu correo electrónico - Socios3"
	body := fmt.Sprintf(`
		<h2>Bienvenido a Socios3</h2>
		<p>Por favor, confirma tu dirección de correo electrónico haciendo clic en el siguiente enlace:</p>
		<p><a href="%s" style="display: inline-block; padding: 10px 20px; color: #fff; background-color: #2563eb; text-decoration: none; border-radius: 5px;">Confirmar Correo</a></p>
		<p>O copia y pega esta URL en tu navegador:</p>
		<p>%s</p>
		<p>Este enlace expirará en 24 horas.</p>
	`, link, link)
	return SendMail(toEmail, subject, body)
}

// SendPasswordRecoveryEmail envía el correo de restablecimiento de contraseña.
func SendPasswordRecoveryEmail(baseURL, toEmail, token string) error {
	link := fmt.Sprintf("%s/restablecer-password?token=%s", baseURL, token)
	subject := "Restablece tu contraseña - Socios3"
	body := fmt.Sprintf(`
		<h2>Recuperación de Contraseña</h2>
		<p>Hemos recibido una solicitud para restablecer tu contraseña. Haz clic en el siguiente botón para continuar:</p>
		<p><a href="%s" style="display: inline-block; padding: 10px 20px; color: #fff; background-color: #10b981; text-decoration: none; border-radius: 5px;">Restablecer Contraseña</a></p>
		<p>O copia y pega esta URL en tu navegador:</p>
		<p>%s</p>
		<p>Este enlace expirará en 1 hora. Si no solicitaste esto, puedes ignorar este correo de forma segura.</p>
	`, link, link)
	return SendMail(toEmail, subject, body)
}
