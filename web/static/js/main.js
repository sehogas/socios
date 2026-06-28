// Funciones de utilidad general para el frontend de Socios

document.addEventListener('DOMContentLoaded', () => {
    // Confirmación para acciones destructivas
    const deleteButtons = document.querySelectorAll('.btn-confirm-delete');
    deleteButtons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            if (!confirm('¿Estás seguro de que deseas eliminar este elemento? Esta acción no se puede deshacer.')) {
                e.preventDefault();
            }
        });
    });

    // Confirmación para backups
    const backupButton = document.querySelector('.btn-confirm-backup');
    if (backupButton) {
        backupButton.addEventListener('click', () => {
            alert('Generando copia de seguridad. El archivo .db se descargará automáticamente a continuación.');
        });
    }

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
