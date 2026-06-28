// Funciones de utilidad general para el frontend de Socios utilizando SweetAlert2

document.addEventListener('DOMContentLoaded', () => {
    // Interceptar botones de confirmación de eliminación (.btn-confirm-delete)
    document.addEventListener('click', (e) => {
        const btn = e.target.closest('.btn-confirm-delete');
        if (btn && !btn.dataset.confirmed) {
            e.preventDefault();
            Swal.fire({
                title: '¿Confirmas la eliminación?',
                text: 'Esta acción no se puede deshacer.',
                icon: 'warning',
                showCancelButton: true,
                confirmButtonColor: '#ef4444', // Rojo Peligro
                cancelButtonColor: '#64748b',   // Gris Muted
                confirmButtonText: 'Sí, eliminar',
                cancelButtonText: 'Cancelar'
            }).then((result) => {
                if (result.isConfirmed) {
                    btn.dataset.confirmed = 'true';
                    if (btn.tagName === 'A') {
                        window.location.href = btn.href;
                    } else if (btn.type === 'submit' && btn.form) {
                        btn.form.submit();
                    } else {
                        btn.click();
                    }
                }
            });
        }
    });

    // Confirmación para backups (.btn-confirm-backup)
    const backupButton = document.querySelector('.btn-confirm-backup');
    if (backupButton) {
        backupButton.addEventListener('click', (e) => {
            if (!backupButton.dataset.confirmed) {
                e.preventDefault();
                Swal.fire({
                    title: 'Generando Copia de Seguridad',
                    text: 'El archivo de base de datos (.db) se preparará y descargará automáticamente a continuación.',
                    icon: 'info',
                    confirmButtonColor: '#2563eb', // Azul Primario
                    confirmButtonText: 'Entendido y Descargar'
                }).then((result) => {
                    if (result.isConfirmed) {
                        backupButton.dataset.confirmed = 'true';
                        // Buscar el formulario contenedor para enviarlo
                        const form = backupButton.closest('form');
                        if (form) {
                            form.submit();
                        } else {
                            backupButton.click();
                        }
                    }
                });
            }
        });
    }

    // Interceptar envíos de formularios con confirmación (data-confirm)
    document.addEventListener('submit', (e) => {
        const form = e.target;
        if (form.hasAttribute('data-confirm') && !form.dataset.confirmed) {
            e.preventDefault();
            Swal.fire({
                title: '¿Confirmas esta acción?',
                text: form.getAttribute('data-confirm'),
                icon: 'question',
                showCancelButton: true,
                confirmButtonColor: '#10b981', // Verde Éxito
                cancelButtonColor: '#64748b',
                confirmButtonText: 'Sí, confirmar',
                cancelButtonText: 'Cancelar'
            }).then((result) => {
                if (result.isConfirmed) {
                    form.dataset.confirmed = 'true';
                    form.submit();
                }
            });
        }
    });

    // Interceptar clics en enlaces o elementos no-formulario con confirmación (data-confirm)
    document.addEventListener('click', (e) => {
        const element = e.target.closest('[data-confirm]');
        if (element && element.tagName !== 'FORM' && !element.dataset.confirmed) {
            // Ignorar si es un botón de submit dentro de un formulario (será manejado por el evento submit del form)
            if (element.tagName === 'BUTTON' && element.type === 'submit' && element.form) {
                return;
            }
            e.preventDefault();
            Swal.fire({
                title: '¿Confirmas esta acción?',
                text: element.getAttribute('data-confirm'),
                icon: 'question',
                showCancelButton: true,
                confirmButtonColor: '#2563eb', // Azul Primario
                cancelButtonColor: '#64748b',
                confirmButtonText: 'Sí, confirmar',
                cancelButtonText: 'Cancelar'
            }).then((result) => {
                if (result.isConfirmed) {
                    element.dataset.confirmed = 'true';
                    if (element.tagName === 'A') {
                        window.location.href = element.href;
                    } else {
                        element.click();
                    }
                }
            });
        }
    });

    // Cerrar alertas automáticamente después de 5 segundos
    const alerts = document.querySelectorAll('.alert');
    alerts.forEach(alert => {
        setTimeout(() => {
            alert.style.transition = 'opacity 0.5s ease';
            alert.style.opacity = '0';
            setTimeout(() => alert.remove(), 500);
        }, 5000);
    });
});
