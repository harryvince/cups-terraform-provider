terraform {
  required_providers {
    cups = {
      source = "registry.terraform.io/harryvince/cups"
    }
  }
}

# Set CUPS_ENDPOINT, CUPS_USERNAME and CUPS_PASSWORD before running Terraform.
provider "cups" {}

resource "cups_printer" "office" {
  name        = "poc-office"
  device_uri  = "ipp://printer.local:8000/ipp/print"
  description = "Provider POC printer"
  location    = "Test room"
}
