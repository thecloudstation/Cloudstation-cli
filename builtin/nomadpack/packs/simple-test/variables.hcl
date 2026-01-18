variable "job_name" {
  description = "The name for the job"
  type        = string
  default     = "simple-test-job"
}

variable "image" {
  description = "Docker image to run"
  type        = string
  default     = "nginx:alpine"
}

variable "port" {
  description = "Port to expose"
  type        = number
  default     = 80
}

variable "cpu" {
  description = "CPU allocation"
  type        = number
  default     = 100
}

variable "memory" {
  description = "Memory allocation"
  type        = number
  default     = 128
}
