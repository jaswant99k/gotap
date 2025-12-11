console.log('goTap static file serving is working!');

document.addEventListener('DOMContentLoaded', function() {
    console.log('Page loaded successfully');
    
    // Add a dynamic timestamp
    const container = document.querySelector('.container');
    const timestamp = document.createElement('p');
    timestamp.style.marginTop = '2rem';
    timestamp.style.color = '#666';
    timestamp.style.fontSize = '0.9rem';
    timestamp.textContent = 'Page loaded at: ' + new Date().toLocaleString();
    container.appendChild(timestamp);
});
