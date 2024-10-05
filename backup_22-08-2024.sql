-- MySQL dump 10.13  Distrib 8.0.32, for Win64 (x86_64)
--
-- Host: 202.10.40.143    Database: selarashomeid
-- ------------------------------------------------------
-- Server version	8.0.36-0ubuntu0.22.04.1

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `access`
--

DROP TABLE IF EXISTS `access`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `access` (
  `id` int NOT NULL AUTO_INCREMENT,
  `module` varchar(100) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=108 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `access`
--

LOCK TABLES `access` WRITE;
/*!40000 ALTER TABLE `access` DISABLE KEYS */;
INSERT INTO `access` VALUES (77,'facebook','2024-06-16 00:48:17'),(78,'tiktok','2024-06-16 00:48:20'),(79,'instagram','2024-06-16 00:48:23'),(80,'whatsapp','2024-06-16 00:48:37'),(81,'facebook','2024-06-16 09:46:54'),(82,'facebook','2024-06-16 15:34:22'),(83,'facebook','2024-06-17 14:40:47'),(84,'facebook','2024-06-18 13:44:11'),(85,'whatsapp','2024-06-19 09:54:27'),(86,'whatsapp','2024-06-19 10:41:24'),(87,'instagram','2024-06-21 07:30:19'),(88,'instagram','2024-06-21 09:08:04'),(89,'whatsapp','2024-06-21 12:03:07'),(90,'whatsapp','2024-06-23 16:37:14'),(91,'whatsapp','2024-06-25 19:36:58'),(92,'facebook','2024-06-28 16:58:56'),(93,'facebook','2024-06-28 16:58:57'),(94,'tIpwmQNW','2024-06-28 16:58:57'),(95,'instagram','2024-06-28 16:59:18'),(96,'tiktok','2024-06-28 17:23:48'),(97,'instagram','2024-06-28 21:15:57'),(98,'tiktok','2024-06-29 07:26:56'),(99,'instagram','2024-06-29 08:56:56'),(102,'whatsapp','2024-07-17 02:48:07'),(103,'tiktok','2024-07-17 03:05:52'),(104,'whatsapp','2024-07-17 03:20:39'),(105,'facebook','2024-07-17 03:36:25'),(106,'instagram','2024-07-17 03:48:02'),(107,'instagram','2024-08-06 10:40:54');
/*!40000 ALTER TABLE `access` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin`
--

DROP TABLE IF EXISTS `admin`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `username` varchar(255) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `is_login` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin`
--

LOCK TABLES `admin` WRITE;
/*!40000 ALTER TABLE `admin` DISABLE KEYS */;
INSERT INTO `admin` VALUES (1,'Admin Pertama','admin01@mailsac.com','admin01','$2a$10$2CfxTBl.1ySPX7aUQVhy1uhlQQTV9afAP956qn.mMzOXsw42YBlrG',0),(2,'Admin Kedua','admin02@mailsac.com','admin02','$2a$10$Z2CYhhG8Fz.3aRTyZ5675e8Aath/Kykmzw2sP29B/.68YSRyE/RF6',0),(3,'Admin Ketiga','admin03@mailsac.com','admin03','$2a$10$7Wor.BIu0F165FXSGtZ3HOBt4aoiprDzo6QkISYOI1CMSq6eS.g2a',0);
/*!40000 ALTER TABLE `admin` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `affiliate`
--

DROP TABLE IF EXISTS `affiliate`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `affiliate` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `instagram` varchar(255) DEFAULT NULL,
  `tiktok` varchar(255) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=34 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `affiliate`
--

LOCK TABLES `affiliate` WRITE;
/*!40000 ALTER TABLE `affiliate` DISABLE KEYS */;
INSERT INTO `affiliate` VALUES (32,'Anang Edi Purnawan','anangedipurnawan@gmail.com','082134412871','@jualanrumah_jogja','@jualanrumah_jogja','2024-06-28 23:31:43');
/*!40000 ALTER TABLE `affiliate` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `contact`
--

DROP TABLE IF EXISTS `contact`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `contact` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `phone` varchar(255) DEFAULT NULL,
  `message` text,
  `created_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=68 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `contact`
--

LOCK TABLES `contact` WRITE;
/*!40000 ALTER TABLE `contact` DISABLE KEYS */;
INSERT INTO `contact` VALUES (63,'Ali Nasti','alinasti02@gmail.com','081213656883','Saya dapat kabar dari Iqbal/Rama bahwa saya mengirim lamaran kerja di selarashome.id','2024-06-25 15:43:24'),(64,'Edi','edi.propertysy@gmail.com','08128240206','Assalamu\'alaikum wrwb<br /><br />Bisa di jadwalkan bisa ketemu dgn Pak Rudi Saputra , Saya dengan Pak Edi di Depok,: 08128240206, Terima kasih sbnya','2024-06-28 17:29:11'),(67,'Yusnar Setiyadi','yusnarsetiyadi150403@gmail.com','081398447822','halo saya mau tanya2 dong perihal kpr syariah','2024-08-17 10:37:07');
/*!40000 ALTER TABLE `contact` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `notification`
--

DROP TABLE IF EXISTS `notification`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `notification` (
  `id` int NOT NULL AUTO_INCREMENT,
  `title` varchar(255) DEFAULT NULL,
  `message` varchar(255) DEFAULT NULL,
  `is_read` tinyint(1) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `module` varchar(255) DEFAULT NULL,
  `data_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=53 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `notification`
--

LOCK TABLES `notification` WRITE;
/*!40000 ALTER TABLE `notification` DISABLE KEYS */;
INSERT INTO `notification` VALUES (46,'Customer (Ali Nasti) give your message','Click to see details',1,'2024-06-25 15:43:24','contact','63'),(47,'Customer (Edi) give your message','Click to see details',1,'2024-06-28 17:29:11','contact','64'),(49,'Affiliate Request from Anang Edi Purnawan','Click to see details',1,'2024-06-28 23:31:43','affiliate','32'),(51,'Affiliate Request from test','Click to see details',1,'2024-07-07 17:35:37','affiliate','33'),(52,'Customer (Yusnar Setiyadi) give your message','Click to see details',0,'2024-08-17 10:37:07','contact','67');
/*!40000 ALTER TABLE `notification` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2024-08-22  4:17:46
