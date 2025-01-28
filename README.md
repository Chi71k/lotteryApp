# Lottery Application

This is a simple lottery application built with Go. The application allows users to register, login, and view a list of lotteries.

## Features

- User registration
- User login with session management
- Display list of lotteries
- Error handling for incorrect login credentials

## Project Structure

- `main.go`: Entry point of the application.
- `handlers/`: Contains the HTTP handlers for different routes.
  - `auth.go`: Handles user authentication (login and registration).
  - `lottery.go`: Handles the display of lotteries.
- `models/`: Contains the data models.
- `templates/`: Contains the HTML templates for rendering the web pages.
  - `home.html`: Home page template.
  - `login.html`: Login page template.
  - `register.html`: Registration page template.
  - `lotteries.html`: Lotteries page template.
- `static/`: Contains static files like CSS.

## Setup

1. Clone the repository:
    ```sh
    git clone https://github.com/your-username/your-repository.git
    cd your-repository
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

3. Set up the database:
    - Create a PostgreSQL database.
    - Run the SQL scripts in the [db](http://_vscodecontentref_/0) directory to set up the tables.

4. Configure the database connection:
    - Update the database connection settings in [db.go](http://_vscodecontentref_/1).

5. Run the application:
    ```sh
    go run main.go
    ```

## Usage

- Open your browser and navigate to `http://localhost:8080`.
- Register a new user.
- Login with the registered user credentials.
- View the list of lotteries.

## Error Handling

- If the username or password is incorrect during login, an error message will be displayed on the login page.

## Contributing

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-branch`).
3. Make your changes.
4. Commit your changes (`git commit -m 'Add some feature'`).
5. Push to the branch (`git push origin feature-branch`).
6. Open a pull request.

## License

This project is licensed under the MIT License.