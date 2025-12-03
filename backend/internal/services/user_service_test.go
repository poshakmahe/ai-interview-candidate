package services

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/katim/secure-doc-vault/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// Helper function to create a mock database
func newMockDB(t *testing.T) (*database.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	return &database.DB{DB: db}, mock
}

func TestNewUserService(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	service := NewUserService(db)

	if service == nil {
		t.Fatal("NewUserService returned nil")
	}

	if service.db != db {
		t.Error("UserService.db not set correctly")
	}
}

func TestUserService_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	email := "test@example.com"
	password := "securepassword123"
	name := "Test User"

	// Mock: Check if user exists (returns no rows)
	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	// Mock: Insert new user
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(sqlmock.AnyArg(), email, sqlmock.AnyArg(), name, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	user, err := service.Create(email, password, name)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user == nil {
		t.Fatal("Create() returned nil user")
	}

	if user.Email != email {
		t.Errorf("user.Email = %q, want %q", user.Email, email)
	}

	if user.Name != name {
		t.Errorf("user.Name = %q, want %q", user.Name, name)
	}

	if user.ID == uuid.Nil {
		t.Error("user.ID should not be nil UUID")
	}

	// Password should be hashed, not plain text
	if user.Password == password {
		t.Error("user.Password should be hashed, not plain text")
	}

	// Verify password hash is valid
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		t.Error("Password hash is invalid")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_Create_UserExists(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	email := "existing@example.com"
	existingID := uuid.New()

	// Mock: User already exists
	rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "created_at", "updated_at"}).
		AddRow(existingID, email, "hashedpw", "Existing User", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnRows(rows)

	_, err := service.Create(email, "password", "New Name")

	if err != ErrUserExists {
		t.Errorf("Create() error = %v, want ErrUserExists", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_Create_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	email := "test@example.com"
	dbError := errors.New("database connection error")

	// Mock: User check fails with DB error
	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnError(dbError)

	_, err := service.Create(email, "password", "Test")

	if err == nil {
		t.Error("Create() should return error on DB failure")
	}
}

func TestUserService_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	userID := uuid.New()
	email := "test@example.com"
	name := "Test User"
	createdAt := time.Now()
	updatedAt := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "created_at", "updated_at"}).
		AddRow(userID, email, "hashedpassword", name, createdAt, updatedAt)

	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE id = \$1`).
		WithArgs(userID).
		WillReturnRows(rows)

	user, err := service.GetByID(userID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if user.ID != userID {
		t.Errorf("user.ID = %v, want %v", user.ID, userID)
	}

	if user.Email != email {
		t.Errorf("user.Email = %q, want %q", user.Email, email)
	}

	if user.Name != name {
		t.Errorf("user.Name = %q, want %q", user.Name, name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	userID := uuid.New()

	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE id = \$1`).
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	_, err := service.GetByID(userID)

	if err != ErrUserNotFound {
		t.Errorf("GetByID() error = %v, want ErrUserNotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_GetByEmail_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	userID := uuid.New()
	email := "test@example.com"
	name := "Test User"

	rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "created_at", "updated_at"}).
		AddRow(userID, email, "hashedpassword", name, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnRows(rows)

	user, err := service.GetByEmail(email)
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	if user.Email != email {
		t.Errorf("user.Email = %q, want %q", user.Email, email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_GetByEmail_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	email := "nonexistent@example.com"

	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	_, err := service.GetByEmail(email)

	if err != ErrUserNotFound {
		t.Errorf("GetByEmail() error = %v, want ErrUserNotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_Authenticate_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	email := "test@example.com"
	password := "correctpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "created_at", "updated_at"}).
		AddRow(userID, email, string(hashedPassword), "Test User", time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnRows(rows)

	user, err := service.Authenticate(email, password)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if user.Email != email {
		t.Errorf("user.Email = %q, want %q", user.Email, email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_Authenticate_WrongPassword(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	email := "test@example.com"
	correctPassword := "correctpassword"
	wrongPassword := "wrongpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)

	rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "created_at", "updated_at"}).
		AddRow(uuid.New(), email, string(hashedPassword), "Test User", time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnRows(rows)

	_, err := service.Authenticate(email, wrongPassword)

	if err != ErrInvalidPassword {
		t.Errorf("Authenticate() with wrong password error = %v, want ErrInvalidPassword", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_Authenticate_UserNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	email := "nonexistent@example.com"

	mock.ExpectQuery(`SELECT id, email, password, name, created_at, updated_at FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	_, err := service.Authenticate(email, "anypassword")

	if err != ErrUserNotFound {
		t.Errorf("Authenticate() with non-existent user error = %v, want ErrUserNotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_UpdateName_Success(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	userID := uuid.New()
	newName := "Updated Name"

	mock.ExpectExec(`UPDATE users SET name = \$1, updated_at = \$2 WHERE id = \$3`).
		WithArgs(newName, sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := service.UpdateName(userID, newName)
	if err != nil {
		t.Fatalf("UpdateName() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_UpdateName_UserNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	userID := uuid.New()
	newName := "Updated Name"

	mock.ExpectExec(`UPDATE users SET name = \$1, updated_at = \$2 WHERE id = \$3`).
		WithArgs(newName, sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	err := service.UpdateName(userID, newName)

	if err != ErrUserNotFound {
		t.Errorf("UpdateName() with non-existent user error = %v, want ErrUserNotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserService_UpdateName_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	service := NewUserService(db)

	userID := uuid.New()
	dbError := errors.New("database error")

	mock.ExpectExec(`UPDATE users SET name = \$1, updated_at = \$2 WHERE id = \$3`).
		WithArgs("New Name", sqlmock.AnyArg(), userID).
		WillReturnError(dbError)

	err := service.UpdateName(userID, "New Name")

	if err == nil {
		t.Error("UpdateName() should return error on DB failure")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// Benchmark tests
func BenchmarkPasswordHashing(b *testing.B) {
	password := "testpassword123"
	for i := 0; i < b.N; i++ {
		bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	}
}

func BenchmarkPasswordVerification(b *testing.B) {
	password := "testpassword123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	}
}
