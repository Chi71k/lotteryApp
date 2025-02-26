document.addEventListener('DOMContentLoaded', function() {
    const cardNumberInput = document.getElementById('card_number');
    const expiryInput = document.getElementById('expiry');
    const cvvInput = document.getElementById('cvv');
  
    if (cardNumberInput) {
      cardNumberInput.addEventListener('input', formatCardNumber);
    }
    if (expiryInput) {
      expiryInput.addEventListener('input', formatExpiry);
    }
    if (cvvInput) {
      cvvInput.addEventListener('input', formatCVV);
    }
  });
  

  
  function formatCardNumber(e) {
    let value = e.target.value;
    value = value.replace(/\D/g, '');
    const chunks = value.match(/.{1,4}/g);
    if (chunks) {
      value = chunks.join(' ');
    }
    e.target.value = value;
  }
  
 
  function formatExpiry(e) {
    let value = e.target.value;
    value = value.replace(/\D/g, '');
    if (value.length > 2) {
      value = value.slice(0, 2) + '/' + value.slice(2);
    }
    if (value.length > 5) {
      value = value.slice(0, 5);
    }
    e.target.value = value;
  }
  
  function formatCVV(e) {
    let value = e.target.value;
    // Удаляем нецифры
    value = value.replace(/\D/g, '');
    // Ограничиваем до 3 символов
    if (value.length > 3) {
      value = value.slice(0, 3);
    }
    e.target.value = value;
  }
  
 
  function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.style.display = "block";
    }
  }
  
  function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.style.display = "none";
    }
  }
  
  window.onclick = function(event) {
    const modals = document.getElementsByClassName("modal");
    for (let i = 0; i < modals.length; i++) {
      if (event.target === modals[i]) {
        modals[i].style.display = "none";
      }
    }
  };
  
 
  function validateRegisterForm() {
    const errorElement = document.getElementById("registerError");
    const password = document.getElementById("reg-password").value;
  
    errorElement.style.display = "none";
    errorElement.textContent = "";
  
    if (password.length < 8) {
      errorElement.textContent = "Password must be at least 8 characters long.";
      errorElement.style.display = "block";
      return false;
    }
  
    if (!/[A-Z]/.test(password)) {
      errorElement.textContent = "Password must contain at least one uppercase letter.";
      errorElement.style.display = "block";
      return false;
    }
  
    if (!/\d/.test(password)) {
      errorElement.textContent = "Password must contain at least one digit.";
      errorElement.style.display = "block";
      return false;
    }
  
    return true;
  }
  

  function validateCardForm() {
    const cardNumber = document.getElementById("card_number").value.replace(/\s/g, '');
    const expiry = document.getElementById("expiry").value.trim();
    const cvv = document.getElementById("cvv").value.trim();
    const amountStr = document.getElementById("amount").value.trim();
  
    if (!/^\d{16}$/.test(cardNumber)) {
      alert("Invalid card number. Must be exactly 16 digits.");
      return false;
    }
  
    if (!/^(0[1-9]|1[0-2])\/\d{2}$/.test(expiry)) {
      alert("Invalid expiry date. Format must be MM/YY, e.g. 08/25.");
      return false;
    }
  
    const now = new Date();
    const currentYear = now.getFullYear() % 100; // две последние цифры
    const currentMonth = now.getMonth() + 1;
    const month = parseInt(expiry.slice(0, 2), 10);
    const year = parseInt(expiry.slice(3), 10);
  
    if (month < 1 || month > 12) {
      alert("Invalid month. Must be between 01 and 12.");
      return false;
    }
    if (year < currentYear || (year === currentYear && month < currentMonth)) {
      alert("Expiry date is in the past.");
      return false;
    }
  
    if (!/^\d{3}$/.test(cvv)) {
      alert("Invalid CVV. Must be 3 digits.");
      return false;
    }
  
    const amount = parseFloat(amountStr);
    if (isNaN(amount) || amount <= 0) {
      alert("Invalid top-up amount.");
      return false;
    }
  
    return true;
  }
  