package main

import (
	"fmt"
	"log"

	"github.com/mxschmitt/playwright-go"
)

func main() {
	// Playwright'ni ishga tushiramiz
	pw, err := playwright.Run()
	if err != nil {
		log.Fatal("Playwright xatosi:", err)
	}
	defer pw.Stop()

	// Browserni ko'rinadigan holatda ochamiz
	browser, err := pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(false),
		},
	)
	if err != nil {
		log.Fatal("Browser xatosi:", err)
	}
	defer browser.Close()

	// Yangi page
	page, err := browser.NewPage()
	if err != nil {
		log.Fatal("Page xatosi:", err)
	}

	// 1. Saytni ochamiz
	_, err = page.Goto("https://web.telegram.org/k/")
	if err != nil {
		log.Fatal("Saytni ochib bo'lmadi:", err)
	}

	fmt.Println("✅ Sayt ochildi")

	// 2. "By phone number" tugmasini topamiz
	phoneButton := page.GetByText("By phone number")

	err = phoneButton.WaitFor()
	if err != nil {
		log.Fatal("'By phone number' topilmadi:", err)
	}

	// 3. Tugmani bosamiz
	err = phoneButton.Click()
	if err != nil {
		log.Fatal("'By phone number' bosilmadi:", err)
	}

	fmt.Println("✅ By phone number bosildi")

	// 4. Telefon inputini topamiz
	phoneInput := page.Locator("#phone")

	err = phoneInput.WaitFor()
	if err != nil {
		log.Fatal("Telefon inputi topilmadi:", err)
	}

	// 5. Telefon raqamini kiritamiz
	err = phoneInput.Fill("909079949")
	if err != nil {
		log.Fatal("Telefon raqami kiritilmadi:", err)
	}

	fmt.Println("✅ Telefon raqami kiritildi")

	// 6. Next tugmasini topamiz
	nextButton := page.GetByRole(
		"button",
		playwright.PageGetByRoleOptions{
			Name: "Next",
		},
	)

	err = nextButton.WaitFor()
	if err != nil {
		log.Fatal("Next tugmasi topilmadi:", err)
	}

	// 7. Next tugmasini bosamiz
	err = nextButton.Click()
	if err != nil {
		log.Fatal("Next bosilmadi:", err)
	}

	fmt.Println("✅ Next bosildi")

	// 8. Keyingi oynani kutamiz
	page.WaitForTimeout(2000)

	fmt.Println("🔎 Keyingi oyna ochildi")

	// 9. Kod inputini topishga harakat qilamiz
	codeInput := page.Locator("#code")

	err = codeInput.WaitFor()
	if err != nil {
		fmt.Println("❌ #code input topilmadi")
		fmt.Println("Keyingi bosqichdagi input boshqa selectorga ega.")
		fmt.Println("Brauzer ochiq qoldirildi.")
		select {}
	}

	fmt.Println("✅ Kod inputi topildi")

	// Test uchun bitta kod
	err = codeInput.Fill("12345")
	if err != nil {
		log.Fatal("Kod kiritilmadi:", err)
	}

	fmt.Println("✅ 12345 kiritildi")

	// Brauzerni ochiq qoldiramiz
	fmt.Println("Dastur ishlashda davom etmoqda...")
	select {}
}
