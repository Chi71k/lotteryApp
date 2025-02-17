// Функции для открытия/закрытия модалок
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
  
  // Закрытие модалки по клику вне её содержимого
  window.onclick = function(event) {
    const modals = document.getElementsByClassName("modal");
    for (let i = 0; i < modals.length; i++) {
      if (event.target === modals[i]) {
        modals[i].style.display = "none";
      }
    }
  }
  
  // Функция валидации пароля при регистрации
  function validateRegisterForm() {
    const errorElement = document.getElementById("registerError");
    const password = document.getElementById("reg-password").value;
  
    // Сбрасываем предыдущее состояние
    errorElement.style.display = "none";
    errorElement.textContent = "";
  
    // 1. Минимум 8 символов
    if (password.length < 8) {
      errorElement.textContent = "Password must be at least 8 characters long.";
      errorElement.style.display = "block";
      return false;
    }
  
    // 2. Хотя бы одна заглавная буква
    if (!/[A-Z]/.test(password)) {
      errorElement.textContent = "Password must contain at least one uppercase letter.";
      errorElement.style.display = "block";
      return false;
    }
  
    // 3. Хотя бы одна цифра
    if (!/\d/.test(password)) {
      errorElement.textContent = "Password must contain at least one digit.";
      errorElement.style.display = "block";
      return false;
    }
  
    // Если все проверки пройдены — отправляем форму
    return true;
  }
  
  // Открыть модалку
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.style.display = "block";
    }
  }
  
  // Закрыть модалку
  function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.style.display = "none";
    }
  }
  
  // Закрытие модалки по клику на подложку
  window.onclick = function(event) {
    const modals = document.getElementsByClassName("modal");
    for (let i = 0; i < modals.length; i++) {
      if (event.target === modals[i]) {
        modals[i].style.display = "none";
      }
    }
  }
  
  // Проверка полей карты
  function validateCardForm() {
    const cardNumber = document.getElementById("card_number").value.trim();
    const expiry = document.getElementById("expiry").value.trim();
    const cvv = document.getElementById("cvv").value.trim();
    const amountStr = document.getElementById("amount").value.trim();
  
    // 1. 16-значный номер
    const cardRegex = /^\d{16}$/;
    if (!cardRegex.test(cardNumber)) {
      alert("Invalid card number. Must be exactly 16 digits.");
      return false;
    }
  
    // 2. Срок действия (MM/YY)
    const expiryRegex = /^(0[1-9]|1[0-2])\/\d{2}$/;
    if (!expiryRegex.test(expiry)) {
      alert("Invalid expiry date. Format must be MM/YY, e.g. 08/25.");
      return false;
    }
  
    // 3. CVV: 3 цифры
    const cvvRegex = /^\d{3}$/;
    if (!cvvRegex.test(cvv)) {
      alert("Invalid CVV. Must be 3 digits.");
      return false;
    }
  
    // 4. Сумма > 0
    const amount = parseFloat(amountStr);
    if (isNaN(amount) || amount <= 0) {
      alert("Invalid top-up amount.");
      return false;
    }
  
    // Все проверки пройдены
    return true;
  }
  