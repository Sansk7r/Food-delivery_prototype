package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	//importing http requests
	//GET - read/search , , , POST - write/append , , , PATCH - update/delete/change status
)

type User struct { //Authentication for login/signup
	Username string `json:"username"`
	Password string `json:"password"`
}

var Users = []User{
	{
		Username: "user1",
		Password: "password1",
	},
	{
		Username: "user2",
		Password: "password2",
	},
}

type Menu struct { //base menu for any restaurant
	Breakfast   RestMenu `json:"breakfast"`
	Lunch       RestMenu `json:"lunch"`
	Dinner      RestMenu `json:"dinner"`
	DeliverFrom int      `json:"delivery from(hrs)"`
	DeliverTill int      `json:"delivery till(hrs)"`
}

type RestMenu struct { //per restaurant
	Dish     string `json:"dish"`
	FoodType string `json:"foodtype"`
	Fav      bool   `json:"favourite"`
}

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type Restaurant struct {
	Name     string  `json:"name"`
	IsClosed bool    `json:"status"` //status for open or closed - deliveries assigned if open
	Addr     Address `json:"address"`
	Menu     Menu    `json:"menu"`
}

type Cart struct { //implement add/view card and placing order
	UserID     string      `json:"user_id"`
	Items      []OrderItem `json:"items"`
	TotalPrice float64     `json:"total_price"`
}

var userCarts = make(map[string]*Cart)

type Order struct {
	OrderID        string         `json:"order_id"`
	Restaurant     Restaurant     `json:"restaurant"`
	Items          []OrderItem    `json:"items"`
	TotalPrice     float64        `json:"total_price"`
	Status         string         `json:"status"`
	DeliveryPerson DeliveryPerson `json:"delivery_person"`
}

type OrderItem struct { //order to be written in besides adding/viewing cart as finalized purchase
	Dish     string  `json:"dish"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

var Orders = []Order{}

type OrderRequest struct {
	RestaurantName string      `json:"restaurant_name"`
	Items          []OrderItem `json:"items"`
}

type OrderResponse struct {
	OrderID string `json:"order_id"`
}

var nextOrderID int = 1 //iterate with respect to available delivery persons so that each person is given an order

var listofRestaurants = []Restaurant{
	{
		Name:     "abc",
		IsClosed: false,
		Addr: Address{
			Street: "street1",
			City:   "city1",
		},
		Menu: Menu{
			Breakfast: RestMenu{
				Dish:     "apple",
				FoodType: "veg",
				Fav:      false,
			},
			Lunch: RestMenu{
				Dish:     "burger",
				FoodType: "non-veg",
				Fav:      true,
			},
			Dinner: RestMenu{
				Dish:     "pizza",
				FoodType: "veg",
				Fav:      true,
			},
			DeliverFrom: 1000,
			DeliverTill: 1600,
		},
	},
	{
		Name:     "xyz",
		IsClosed: true,
		Addr: Address{
			Street: "street2",
			City:   "city2",
		},
		Menu: Menu{
			Breakfast: RestMenu{
				Dish:     "cereal",
				FoodType: "vegan",
				Fav:      false,
			},
			Lunch: RestMenu{
				Dish:     "salad",
				FoodType: "vegan",
				Fav:      false,
			},
			Dinner: RestMenu{
				Dish:     "soy",
				FoodType: "vegan",
				Fav:      false,
			}, DeliverFrom: 1300,
			DeliverTill: 1900,
		},
	},
}

type DeliveryPerson struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	IsAvailable bool   `json:"is_available"`
}

var deliveryPersons = []DeliveryPerson{
	{
		ID:          1,
		Name:        "John Doe",
		PhoneNumber: "123-456-7890",
		IsAvailable: true,
	},
	{
		ID:          2,
		Name:        "Jane Smith",
		PhoneNumber: "987-654-3210",
		IsAvailable: true,
	},
}

func getAllDetails(c *gin.Context) { //public function to get all details
	c.IndentedJSON(http.StatusOK, listofRestaurants)
}

func authenticateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("API-Key")
		validAPIKey := "key"
		if apiKey != validAPIKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func registerUser(c *gin.Context) {
	var newUser User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user data"})
		return
	}
	for _, existingUser := range Users {
		if existingUser.Username == newUser.Username {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
	}
	Users = append(Users, newUser)
	c.Status(http.StatusCreated)
}

func loginUser(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login data"})
		return
	}

	for _, user := range Users {
		if user.Username == credentials.Username && user.Password == credentials.Password {
			token, err := generateToken(credentials.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"token": token, "message": "Login successful"})
			return
		}
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			c.Abort()
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token"})
			c.Abort()
			return
		}

		if token.Valid {
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token"})
			c.Abort()
		}
	}
}

func searchFoodItem(c *gin.Context) {
	foodItem := c.Query("fooditem")
	matchingRestaurants := []Restaurant{}
	for _, restaurant := range listofRestaurants {
		if containsFoodItem(restaurant, foodItem) {
			matchingRestaurants = append(matchingRestaurants, restaurant)
		}
	}
	c.IndentedJSON(http.StatusOK, matchingRestaurants)
}

func containsFoodItem(restaurant Restaurant, foodItem string) bool {
	return strings.Contains(strings.ToLower(restaurant.Menu.Breakfast.Dish), strings.ToLower(foodItem)) ||
		strings.Contains(strings.ToLower(restaurant.Menu.Lunch.Dish), strings.ToLower(foodItem)) ||
		strings.Contains(strings.ToLower(restaurant.Menu.Dinner.Dish), strings.ToLower(foodItem))
}

func getMenu(c *gin.Context) {
	restaurantName := c.Param("name")
	var foundRestaurant *Restaurant
	for _, restaurant := range listofRestaurants {
		if restaurant.Name == restaurantName {
			foundRestaurant = &restaurant
			break
		}
	}
	if foundRestaurant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, foundRestaurant.Menu)
}

func getFoodByType(c *gin.Context) {
	foodType := c.Param("foodtype")
	matchingFoods := []RestMenu{}
	for _, restaurant := range listofRestaurants {
		if strings.EqualFold(restaurant.Menu.Breakfast.FoodType, foodType) {
			matchingFoods = append(matchingFoods, restaurant.Menu.Breakfast)
		}
		if strings.EqualFold(restaurant.Menu.Lunch.FoodType, foodType) {
			matchingFoods = append(matchingFoods, restaurant.Menu.Lunch)
		}
		if strings.EqualFold(restaurant.Menu.Dinner.FoodType, foodType) {
			matchingFoods = append(matchingFoods, restaurant.Menu.Dinner)
		}
	}
	if len(matchingFoods) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No foods found with the specified FoodType"})
		return
	}

	c.IndentedJSON(http.StatusOK, matchingFoods)
}

func deliveryPersonStatusUpdate(c *gin.Context) { //change status of delivery person for order, if any - a person allocated is unavailable for other deliveries
	restaurantName := c.Param("name")
	dish := c.Param("dish")

	var foundRestaurant *Restaurant
	for i, restaurant := range listofRestaurants {
		if restaurant.Name == restaurantName {
			foundRestaurant = &listofRestaurants[i]
			break
		}
	}

	if foundRestaurant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	switch {
	case strings.EqualFold(foundRestaurant.Menu.Breakfast.Dish, dish):
		foundRestaurant.Menu.Breakfast.Fav = !foundRestaurant.Menu.Breakfast.Fav
	case strings.EqualFold(foundRestaurant.Menu.Lunch.Dish, dish):
		foundRestaurant.Menu.Lunch.Fav = !foundRestaurant.Menu.Lunch.Fav
	case strings.EqualFold(foundRestaurant.Menu.Dinner.Dish, dish):
		foundRestaurant.Menu.Dinner.Fav = !foundRestaurant.Menu.Dinner.Fav
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dish not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, foundRestaurant)
}

func showFavourite(c *gin.Context) {
	matchingRestaurants := []Restaurant{}

	for _, restaurant := range listofRestaurants {
		if restaurant.Menu.Breakfast.Fav || restaurant.Menu.Lunch.Fav || restaurant.Menu.Dinner.Fav {
			matchingRestaurants = append(matchingRestaurants, restaurant)
		}
	}

	c.IndentedJSON(http.StatusOK, matchingRestaurants)
}

func addToCart(c *gin.Context) {
	userID := c.Param("user_id")
	var item OrderItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item data"})
		return
	}
	cart, exists := userCarts[userID]
	if !exists {
		cart = &Cart{
			UserID: userID,
			Items:  make([]OrderItem, 0),
		}
	}

	cart.Items = append(cart.Items, item)
	cart.TotalPrice += item.Price * float64(item.Quantity)
	userCarts[userID] = cart
	c.JSON(http.StatusCreated, cart)
}

func viewCart(c *gin.Context) {
	userID := c.Param("user_id")
	cart, exists := userCarts[userID]
	if !exists {
		c.JSON(http.StatusOK, &Cart{UserID: userID, Items: []OrderItem{}, TotalPrice: 0})
		return
	}
	c.JSON(http.StatusOK, cart)
}

func getAvailableDeliveryPerson() DeliveryPerson {
	for _, dp := range deliveryPersons {
		if dp.IsAvailable {
			return dp
		}
	}
	return DeliveryPerson{
		ID:          0,
		Name:        "No Delivery Person Available",
		PhoneNumber: "No Phone Number Available",
		IsAvailable: false,
	}
}

func placeOrder(c *gin.Context) {
	var orderRequest OrderRequest
	if err := c.ShouldBindJSON(&orderRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order data"})
		return
	}
	var selectedRestaurant *Restaurant
	for _, restaurant := range listofRestaurants {
		if restaurant.Name == orderRequest.RestaurantName {
			selectedRestaurant = &restaurant
			break
		}
	}

	if selectedRestaurant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	deliveryPerson := getAvailableDeliveryPerson()
	if deliveryPerson.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No available delivery person"})
		return
	}

	for i, dp := range deliveryPersons {
		if dp.ID == deliveryPerson.ID {
			deliveryPersons[i].IsAvailable = false
			break
		}
	}

	totalPrice := 0.0
	for _, item := range orderRequest.Items {
		totalPrice += float64(item.Quantity) * item.Price
	}

	newOrder := Order{
		OrderID:        fmt.Sprintf("ORDER-%d", nextOrderID),
		Restaurant:     *selectedRestaurant,
		Items:          orderRequest.Items,
		TotalPrice:     totalPrice,
		Status:         "Out for Delivery",
		DeliveryPerson: deliveryPerson,
	}
	Orders = append(Orders, newOrder)
	nextOrderID++
	c.JSON(http.StatusCreated, OrderResponse{OrderID: newOrder.OrderID})
}

func getOrderStatus(c *gin.Context) {
	orderID := c.Param("order_id")
	var foundOrder *Order
	for _, order := range Orders {
		if order.OrderID == orderID {
			foundOrder = &order
			break
		}
	}

	if foundOrder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	var deliveryStatus string

	if foundOrder.DeliveryPerson.ID == 0 {
		deliveryStatus = "Pending"
	} else {
		deliveryStatus = "Out for Delivery"
	}

	response := gin.H{
		"total_price": foundOrder.TotalPrice,
		"status":      deliveryStatus,
		"delivery_person": gin.H{
			"name":         foundOrder.DeliveryPerson.Name,
			"phone_number": foundOrder.DeliveryPerson.PhoneNumber,
		},
	}

	c.IndentedJSON(http.StatusOK, response)
}

//mongoDB server creation - http requests are ambiguous and sometimes are not returning output - hence commented out
//requests are still functional over localhost

/*type App struct {
	MongoClient *mongo.Client
}

func (app *App) Initialize() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	app.MongoClient = client

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("app", app)
		c.Next()
	})

	router.GET("/listofRestaurants", app.getAllRestaurants)
	router.POST("/addRestaurant", app.addRestaurant)
	router.GET("/search", app.searchFoodItem)
	router.Run("localhost:8080")
}

func (app *App) getAllRestaurants(c *gin.Context) {
	client := app.MongoClient
	collection := client.Database("mydb").Collection("restaurants")

	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve restaurants"})
		return
	}
	defer cur.Close(context.Background())

	var restaurants []Restaurant
	for cur.Next(context.Background()) {
		var restaurant Restaurant
		if err := cur.Decode(&restaurant); err != nil {
			log.Println(err)
			continue
		}
		restaurants = append(restaurants, restaurant)
	}

	c.JSON(http.StatusOK, restaurants)
}

func (app *App) addRestaurant(c *gin.Context) {
	var restaurant Restaurant
	if err := c.ShouldBindJSON(&restaurant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid restaurant data"})
		return
	}

	client := app.MongoClient
	collection := client.Database("mydb").Collection("restaurants")

	_, err := collection.InsertOne(context.Background(), restaurant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert restaurant"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Restaurant added successfully"})
}

func (app *App) searchFoodItem(c *gin.Context) {
	client := app.MongoClient
	collection := client.Database("mydb").Collection("restaurants")

	foodItem := c.Query("fooditem")
	filter := bson.M{
		"$or": []bson.M{
			{"menu.breakfast.dish": bson.M{"$regex": primitive.Regex{Pattern: foodItem, Options: "i"}}},
			{"menu.lunch.dish": bson.M{"$regex": primitive.Regex{Pattern: foodItem, Options: "i"}}},
			{"menu.dinner.dish": bson.M{"$regex": primitive.Regex{Pattern: foodItem, Options: "i"}}},
		},
	}

	cur, err := collection.Find(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search for restaurants"})
		return
	}
	defer cur.Close(context.Background())

	var matchingRestaurants []Restaurant
	for cur.Next(context.Background()) {
		var restaurant Restaurant
		if err := cur.Decode(&restaurant); err != nil {
			log.Println(err)
			continue
		}
		matchingRestaurants = append(matchingRestaurants, restaurant)
	}

	c.JSON(http.StatusOK, matchingRestaurants)
}*/

var jwtSecret = []byte("your-secret-key") //token based authentication

func generateToken(username string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = username
	claims["exp"] = time.Now().Add(time.Hour).Unix() //setting a time following which token expires

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

//Token generated using login endpoint -> copy token and paste in Header value for key 'Authorization'
//If no token is generated -> user must first register and then attempt to login
/*
	3 Headers to be implemented in <Key,Value> pairs:
	Content-Type	=	application/json.
	API-Key	=	key
	Authorization = <token generated via post request from login endpoint>
*/

func main() {

	// implementing mongoDB server - http requests take time to generate response and sometimes the requests return error - hence they have been commented out

	/*app := App{}
	app.Initialize()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())*/

	router := gin.Default()
	router.Use(func(c *gin.Context) { // -> Headers -> API-Key = key
		//c.Set("mongoClient", client)
		if c.Request.URL.Path != "/listofRestaurants" {
			authenticateAPIKey()(c)
			return
		}
		c.Next()
	})
	router.GET("/listofRestaurants", getAllDetails) // -> /listofRestaurants
	router.POST("/register", registerUser)          // -> /register
	router.POST("/login", loginUser)                // -> /login
	/*{
	    "username": "newuser",
	    "password": "newpassword"
	}*/
	router.GET("/search", searchFoodItem)                             // -> /search?fooditem=<fooditem>
	router.GET("/menu/:name", getMenu)                                // -> /menu/<resto_name>
	router.GET("/foodtype/:foodtype", getFoodByType)                  // -> /foodtype/<foodtype>
	router.GET("/favsearch", showFavourite)                           // -> /favsearch
	router.PATCH("/update/:name/:dish", deliveryPersonStatusUpdate)   // -> /update/<restaurant_name/<fooditem>
	router.POST("/add-to-cart/:user_id", authMiddleware(), addToCart) // -> /add-to-cart/user1
	/*{
	    "dish": "pizza",
	    "quantity": 2,
	    "price": 10.99
	}*/
	router.GET("/view-cart/:user_id", authMiddleware(), viewCart) // -> /view-cart/user1
	router.POST("/order", authMiddleware(), placeOrder)           // -> /order
	/*{{
	    "restaurant_name": "abc",
	    "items": [
	        {
	            "dish": "apple",
	            "quantity": 2,
	            "price": 5.99
	        },
	        {
	            "dish": "burger",
	            "quantity": 2,
	            "price": 7.99
	        }
	    ]
	}*/
	router.GET("/order/:order_id", authMiddleware(), getOrderStatus) // -> /order/<order_id>
	router.Run("localhost:8080")
}

// All of the above end points are functional and display required output - verified on postman
