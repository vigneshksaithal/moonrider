import 'package:flutter/material.dart';

/// Identity observed on moonrider.ai. See docs/BRANDING.md for asset sources.
const brandOrange = Color(0xFFE7920D);
const brandInk = Color(0xFF212121);
const appBackground = Color(0xFFF7F7F7);
const muted = Color(0xFF5D5D5D);
const warning = Color(0xFFB42318);
const success = Color(0xFF267445);
const brandTint = Color(0xFFFFF4E4);

const pageTitleStyle = TextStyle(
  fontFamily: 'Orbitron',
  fontSize: 26,
  fontWeight: FontWeight.w600,
  color: brandInk,
);

ThemeData moonriderTheme() => ThemeData(
  useMaterial3: true,
  fontFamily: 'Poppins',
  scaffoldBackgroundColor: appBackground,
  colorScheme: ColorScheme.fromSeed(
    seedColor: brandOrange,
    primary: brandOrange,
    onPrimary: brandInk,
    secondary: brandInk,
    surface: Colors.white,
    onSurface: brandInk,
    error: warning,
  ),
  appBarTheme: const AppBarTheme(
    backgroundColor: Colors.white,
    surfaceTintColor: Colors.transparent,
    toolbarHeight: 80,
    titleSpacing: 24,
  ),
  cardTheme: CardThemeData(
    color: Colors.white,
    elevation: 0,
    margin: EdgeInsets.zero,
    shape: RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(12),
      side: const BorderSide(color: Color(0xFFE4E4E4)),
    ),
  ),
  listTileTheme: const ListTileThemeData(
    textColor: brandInk,
    subtitleTextStyle: TextStyle(
      color: muted,
      fontFamily: 'Poppins',
      fontSize: 12,
    ),
  ),
  navigationBarTheme: const NavigationBarThemeData(
    backgroundColor: Colors.white,
    indicatorColor: brandOrange,
  ),
);
